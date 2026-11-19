// Package core runs mihomo (Clash.Meta) in the app's process: it merges the
// app's settings into a profile's configuration, applies it, and exposes
// the core's logs.
package core

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/metacubex/mihomo/component/profile/cachefile"
	"github.com/metacubex/mihomo/config"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/hub"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/hub/route"
	"github.com/metacubex/mihomo/log"

	"github.com/victorzhao/clash-go/internal/store"
)

// LogEntry is a line of the core's log.
type LogEntry struct {
	At      time.Time
	Level   string
	Message string
}

// Manager runs the core and keeps its state.
type Manager struct {
	mu      sync.Mutex
	home    string
	running bool
	lastErr string

	logMu  sync.Mutex
	onLog  func(LogEntry)
	logSub <-chan log.Event
}

// New returns a manager whose home is the core's configuration directory.
func New(home string) *Manager {
	m := &Manager{home: home}
	m.watchLogs()
	return m
}

// OnLog sets the function that receives the core's log lines. It runs on
// the goroutine that reads the core's logs.
func (m *Manager) OnLog(fn func(LogEntry)) {
	m.logMu.Lock()
	m.onLog = fn
	m.logMu.Unlock()
}

func (m *Manager) watchLogs() {
	sub := log.Subscribe()
	m.logSub = sub
	go func() {
		for e := range sub {
			m.logMu.Lock()
			fn := m.onLog
			m.logMu.Unlock()
			if fn == nil {
				continue
			}
			fn(LogEntry{At: time.Now(), Level: e.Type(), Message: e.Payload})
		}
	}()
}

// Running reports whether the core runs, and the error that stopped it
// last.
func (m *Manager) Running() (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running, m.lastErr
}

// Start applies the profile's configuration with the app's settings and
// starts the core, or restarts it with the new configuration.
func (m *Manager) Start(profile []byte, s store.Settings) error {
	if len(profile) == 0 {
		return errors.New("配置为空")
	}
	merged, err := Merge(profile, s)
	if err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	C.SetHomeDir(m.home)
	C.SetConfig(filepath.Join(m.home, "config.yaml"))
	if err := config.Init(m.home); err != nil {
		m.lastErr = err.Error()
		return err
	}
	log.SetLevel(parseLevel(s.LogLevel))

	if err := hub.Parse(merged,
		hub.WithExternalController(fmt.Sprintf("127.0.0.1:%d", s.APIPort)),
		hub.WithSecret(s.Secret),
	); err != nil {
		m.running = false
		m.lastErr = err.Error()
		return err
	}
	m.running = true
	m.lastErr = ""
	return nil
}

// Stop stops the core and its API server.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	executor.Shutdown()
	route.ReCreateServer(&route.Config{})
	_ = cachefile.Cache().Close()
	m.running = false
}

// Merge turns a profile into the configuration the core runs: the app's
// settings replace the profile's own for the ports, the mode, the API and
// TUN, and the profile keeps the rest, as its proxies and rules.
func Merge(profile []byte, s store.Settings) ([]byte, error) {
	var cfg map[string]any
	if err := yaml.Unmarshal(profile, &cfg); err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = map[string]any{}
	}

	// The app owns the inbound ports: a profile's own port and socks-port
	// would open listeners the app doesn't show, and clash them with the
	// mixed port.
	for _, key := range []string{"port", "socks-port", "redir-port", "tproxy-port"} {
		delete(cfg, key)
	}

	cfg["mode"] = s.Mode
	cfg["mixed-port"] = s.MixedPort
	cfg["allow-lan"] = s.AllowLAN
	cfg["ipv6"] = s.IPv6
	cfg["log-level"] = s.LogLevel
	cfg["external-controller"] = fmt.Sprintf("127.0.0.1:%d", s.APIPort)
	cfg["secret"] = s.Secret
	cfg["unified-delay"] = true
	cfg["tcp-concurrent"] = true
	if _, ok := cfg["bind-address"]; !ok {
		cfg["bind-address"] = "*"
	}
	// The geo databases come from a CDN rather than GitHub, which many
	// networks reach slowly, unless the profile names its own sources.
	if _, ok := cfg["geox-url"]; !ok {
		cfg["geox-url"] = map[string]any{
			"geoip":   "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.dat",
			"geosite": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geosite.dat",
			"mmdb":    "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/country.mmdb",
			"asn":     "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/GeoLite2-ASN.mmdb",
		}
	}

	// DNS follows the rules, so queries to blocked domains are made over
	// the proxy instead of leaking to the local resolver. mihomo fixed the
	// rule-mode dial misreport, making respect-rules safe to enable: a
	// proxy-server-nameserver is required, and its own address is resolved
	// directly to avoid a loop.
	dns, _ := cfg["dns"].(map[string]any)
	if dns == nil {
		dns = map[string]any{}
	}
	dns["enable"] = true
	if _, ok := dns["nameserver"]; !ok {
		dns["nameserver"] = []any{"https://223.5.5.5/dns-query", "tls://223.5.5.5:853"}
	}
	if _, ok := dns["proxy-server-nameserver"]; !ok {
		dns["proxy-server-nameserver"] = []any{"https://223.5.5.5/dns-query"}
	}
	dns["respect-rules"] = true
	cfg["dns"] = dns

	enableRealityMLKEM(cfg)

	tun, _ := cfg["tun"].(map[string]any)
	if tun == nil {
		tun = map[string]any{}
	}
	tun["enable"] = s.TunEnabled
	if s.TunEnabled {
		stack := s.TunStack
		if stack == "" {
			stack = "mixed"
		}
		tun["stack"] = stack
		tun["dns-hijack"] = []any{"any:53"}
		if _, ok := tun["auto-route"]; !ok {
			tun["auto-route"] = true
		}
		if _, ok := tun["auto-detect-interface"]; !ok {
			tun["auto-detect-interface"] = true
		}
	}
	cfg["tun"] = tun

	return yaml.Marshal(cfg)
}

func parseLevel(level string) log.LogLevel {
	switch level {
	case "debug":
		return log.DEBUG
	case "warning", "warn":
		return log.WARNING
	case "error":
		return log.ERROR
	case "silent":
		return log.SILENT
	default:
		return log.INFO
	}
}

// enableRealityMLKEM lets the proxies using REALITY offer the hybrid
// X25519MLKEM768 key exchange, which newer servers require: their REALITY
// authentication fails when a client offers X25519 alone. A proxy that sets
// support-x25519mlkem768 itself keeps its choice, for the servers that
// predate the hybrid.
func enableRealityMLKEM(cfg map[string]any) {
	proxies, _ := cfg["proxies"].([]any)
	for _, item := range proxies {
		proxy, ok := item.(map[string]any)
		if !ok {
			continue
		}
		opts, ok := proxy["reality-opts"].(map[string]any)
		if !ok {
			continue
		}
		if _, set := opts["support-x25519mlkem768"]; !set {
			opts["support-x25519mlkem768"] = true
		}
	}
}
