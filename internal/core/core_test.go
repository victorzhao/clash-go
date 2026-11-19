package core

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/victorzhao/clash-go/internal/store"
)

// Merge replaces the profile's inbound ports, mode and API with the app's,
// and keeps its proxies and rules.
func TestMerge(t *testing.T) {
	profile := []byte(`mixed-port: 8080
port: 7890
socks-port: 7891
mode: global
proxies:
  - {name: a, type: socks5, server: 127.0.0.1, port: 1080}
  - name: r
    type: vless
    server: 127.0.0.1
    port: 443
    uuid: 00000000-0000-0000-0000-000000000000
    tls: true
    servername: www.microsoft.com
    reality-opts:
      public-key: VEEo7YejpTQ7gC7oC2ua2ZPATgKlmjgpR_no19iwFHY
      short-id: "0a1f2fad4b4533ba"
rules:
  - MATCH,DIRECT
tun:
  enable: false
  stack: gvisor
  auto-route: true
proxies2: []
`)
	s := store.Settings{
		Mode:       "rule",
		MixedPort:  7890,
		APIPort:    6170,
		Secret:     "s3cret",
		LogLevel:   "warning",
		IPv6:       true,
		TunEnabled: true,
		TunStack:   "mixed",
	}
	out, err := Merge(profile, s)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(out, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["mode"] != "rule" {
		t.Errorf("mode = %v, want rule", cfg["mode"])
	}
	if cfg["mixed-port"] != 7890 {
		t.Errorf("mixed-port = %v, want 7890", cfg["mixed-port"])
	}
	if _, ok := cfg["port"]; ok {
		t.Error("port 应被删除")
	}
	if _, ok := cfg["socks-port"]; ok {
		t.Error("socks-port 应被删除")
	}
	if cfg["external-controller"] != "127.0.0.1:6170" {
		t.Errorf("external-controller = %v", cfg["external-controller"])
	}
	if cfg["secret"] != "s3cret" {
		t.Errorf("secret = %v", cfg["secret"])
	}
	if cfg["ipv6"] != true {
		t.Errorf("ipv6 = %v", cfg["ipv6"])
	}
	tun, ok := cfg["tun"].(map[string]any)
	if !ok || tun["enable"] != true || tun["stack"] != "mixed" {
		t.Errorf("tun = %v, want enabled with mixed stack", cfg["tun"])
	}
	proxies, _ := cfg["proxies"].([]any)
	if len(proxies) != 2 {
		t.Errorf("代理数 %d", len(proxies))
	}
	proxy, _ := proxies[1].(map[string]any)
	opts, _ := proxy["reality-opts"].(map[string]any)
	if opts["support-x25519mlkem768"] != true {
		t.Errorf("reality-opts 没有开启后量子密钥交换: %v", opts)
	}
	if _, ok := cfg["rules"]; !ok {
		t.Error("规则被删掉了")
	}
	if _, ok := cfg["geox-url"]; !ok {
		t.Error("缺少 geox-url")
	}
	dns, ok := cfg["dns"].(map[string]any)
	if !ok || dns["enable"] != true || dns["respect-rules"] != true {
		t.Errorf("dns = %v, want enabled with respect-rules", cfg["dns"])
	}
	if ps, ok := dns["proxy-server-nameserver"].([]any); !ok || len(ps) == 0 {
		t.Errorf("proxy-server-nameserver = %v, want non-empty", dns["proxy-server-nameserver"])
	}
}

func TestMergePreservesCustomDNS(t *testing.T) {
	profile := []byte(`dns:
  enable: true
  enhanced-mode: fake-ip
  fake-ip-range: 198.18.0.1/16
  nameserver: [https://doh.pub/dns-query]
proxies: []
rules: [MATCH,DIRECT]
`)
	out, err := Merge(profile, store.Default())
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(out, &cfg); err != nil {
		t.Fatal(err)
	}
	dns := cfg["dns"].(map[string]any)
	if dns["enhanced-mode"] != "fake-ip" || dns["fake-ip-range"] != "198.18.0.1/16" {
		t.Errorf("自定义 DNS 被覆盖：%v", dns)
	}
	if dns["respect-rules"] != true {
		t.Error("respect-rules 未开启")
	}
	if _, ok := dns["proxy-server-nameserver"]; !ok {
		t.Error("缺少 proxy-server-nameserver")
	}
}

func TestMergeRejectsInvalidYAML(t *testing.T) {
	if _, err := Merge([]byte("proxies: [\n"), store.Default()); err == nil {
		t.Error("无效 YAML 应当报错")
	}
}
