package app

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/victorzhao/clash-go/internal/core"
	"github.com/victorzhao/clash-go/internal/mihomoapi"
	"github.com/victorzhao/clash-go/internal/store"
)

const demoProfile = `# 示例订阅配置
mixed-port: 7890
allow-lan: false
mode: rule
log-level: info

proxies:
  - name: 香港 01
    type: ss
    server: hk01.example.com
    port: 443
    cipher: aes-256-gcm
    password: demo
  - name: 日本 02
    type: trojan
    server: jp02.example.com
    port: 443
    password: demo
    sni: jp02.example.com

proxy-groups:
  - name: 节点选择
    type: select
    proxies:
      - 香港 01
      - 日本 02

rules:
  - DOMAIN-SUFFIX,google.com,节点选择
  - MATCH,DIRECT
`

// TestRender draws the pages with data, writing PNGs to a directory of the
// temp folder, for looking at the interface without a window.
func TestRender(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "clashgo-render")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	pages := []string{"/dashboard", "/proxies", "/profiles", "/profiles/demo", "/connections", "/rules", "/logs", "/settings"}
	for _, p := range pages {
		a := demoApp(t, p)
		tt := ui.NewTester(a.View, 1500, 1000)
		tt.Frame()
		name := filepath.Join(dir, strings.ReplaceAll(strings.TrimPrefix(p, "/"), "/", "-")+".png")
		f, err := os.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
		f.Close()
		t.Log(name)
	}
	// The dashboard at the smallest window size the app allows.
	a := demoApp(t, "/dashboard")
	small := ui.NewTester(a.View, 1000, 680)
	small.Frame()
	f, err := os.Create(filepath.Join(dir, "dashboard-small.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, small.Image()); err != nil {
		t.Fatal(err)
	}
	t.Log(filepath.Join(dir, "dashboard-small.png"))

	dark := demoApp(t, "/dashboard")
	dark.settings.Theme = "dark"
	darkTT := ui.NewTester(dark.View, 1200, 780)
	darkTT.Frame()
	df, err := os.Create(filepath.Join(dir, "dashboard-dark.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer df.Close()
	if err := png.Encode(df, darkTT.Image()); err != nil {
		t.Fatal(err)
	}
	t.Log(filepath.Join(dir, "dashboard-dark.png"))
}

func demoApp(t *testing.T, page string) *App {
	t.Helper()
	a, err := NewAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.settings.Theme = "light"
	a.running.Store(true)
	a.startedAt = time.Now().Add(-4 * time.Second)
	a.lanIP = "192.168.3.163"
	a.apiLatency = 288
	a.testOK, a.testDelay = true, 288
	a.appMem = 270 << 20
	a.recent = []closedRequest{
		{Host: "www.youtube.com", Chains: []string{"节点选择", "香港 01"}, Down: 2_400_000, Up: 60_000, At: time.Now().Add(-2 * time.Minute)},
		{Host: "github.com", Chains: []string{"节点选择", "日本 02"}, Down: 320_000, Up: 12_000, At: time.Now().Add(-7 * time.Minute)},
	}
	a.version = "v1.19.34"
	a.settings.ActiveProfile = "demo"
	a.profiles = []store.Profile{{ID: "demo", Name: "机场订阅", Type: "remote", URL: "https://example.com/sub", UpdatedAt: time.Now()}}
	if err := a.st.WriteProfile("demo", []byte(demoProfile)); err != nil {
		t.Fatal(err)
	}
	a.proxies = map[string]*mihomoapi.Proxy{
		"节点选择": {Name: "节点选择", Type: "Selector", Now: "香港 01", All: []string{"香港 01", "日本 02", "新加坡 03", "美国 04"}},
	}
	a.groups = []string{"节点选择"}
	a.proxies["香港 01"] = &mihomoapi.Proxy{Name: "香港 01", Type: "Shadowsocks", Delay: 42}
	a.proxies["日本 02"] = &mihomoapi.Proxy{Name: "日本 02", Type: "Trojan", Delay: 128}
	a.proxies["新加坡 03"] = &mihomoapi.Proxy{Name: "新加坡 03", Type: "Vmess", Delay: 310}
	a.proxies["美国 04"] = &mihomoapi.Proxy{Name: "美国 04", Type: "Hysteria2"}
	for i := 0; i < 60; i++ {
		a.history = append(a.history, sample{up: int64(20_000 + i*900), down: int64(180_000 + i*2400)})
	}
	a.down, a.up = 320_000, 74_000
	a.totalDown, a.totalUp = 3_200_000_000, 420_000_000
	a.memory = 92 << 20
	a.connCount = 3
	a.conns = []mihomoapi.Connection{
		{Metadata: mihomoapi.Metadata{Host: "www.google.com", Type: "HTTP", Network: "tcp", SourceIP: "127.0.0.1", SourcePort: "51234", DstPort: "443"}, Chains: []string{"香港 01"}, Rule: "RuleSet", Payload: "proxy", Download: 1_800_000, Upload: 120_000, Start: time.Now().Add(-4 * time.Minute)},
		{Metadata: mihomoapi.Metadata{Host: "api.github.com", Type: "HTTP", Network: "tcp", SourceIP: "127.0.0.1", SourcePort: "51892", DstPort: "443"}, Chains: []string{"节点选择", "日本 02"}, Rule: "MATCH", Payload: "节点选择", Download: 640_000, Upload: 32_000, Start: time.Now().Add(-90 * time.Second)},
	}
	a.rules = []mihomoapi.Rule{
		{Type: "RuleSet", Payload: "geosite:google", Proxy: "节点选择"},
		{Type: "DomainSuffix", Payload: "github.com", Proxy: "节点选择"},
		{Type: "MATCH", Payload: "", Proxy: "DIRECT"},
	}
	a.logs = []core.LogEntry{
		{At: time.Now().Add(-6 * time.Second), Level: "info", Message: "RESTful API listening at: 127.0.0.1:6170"},
		{At: time.Now().Add(-5 * time.Second), Level: "info", Message: "[TCP] 127.0.0.1:51234 --> www.google.com:443 match RuleSet(geosite:google) using 香港 01"},
		{At: time.Now().Add(-3 * time.Second), Level: "warning", Message: "[DNS] 未找到 GeoIP 数据库，正在下载"},
		{At: time.Now().Add(-1 * time.Second), Level: "error", Message: "[Proxy] 美国 04 connect failed: timeout"},
	}
	a.route.Replace(page)
	return a
}
