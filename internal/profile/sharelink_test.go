package profile

import (
	"encoding/base64"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const vlessLink = "vless://a1b2c3d4-eacc-4433-981b-7e5f9a8b@142.98.76.54:34888?encryption=none&security=reality&type=tcp&sni=github.io&fp=chrome&pbk=ppQ9FwLrLIa0AOrp1WvcyiaQ37vg2WSy_CD4bIdiTUw&sid=6ba85179f3a2b4c5&flow=xtls-rprx-vision#My-VLESS-Reality-Vision"

func TestConvertVlessLink(t *testing.T) {
	if !isShareLink([]byte(vlessLink)) {
		t.Fatal("vless link not detected as share link")
	}
	out, err := Prepare([]byte(vlessLink))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Proxies     []map[string]any `yaml:"proxies"`
		ProxyGroups []map[string]any `yaml:"proxy-groups"`
		Rules       []string         `yaml:"rules"`
	}
	if err := yaml.Unmarshal(out, &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Proxies) != 1 {
		t.Fatalf("proxies = %d, want 1: %s", len(cfg.Proxies), out)
	}
	p := cfg.Proxies[0]
	if p["type"] != "vless" || p["name"] != "My-VLESS-Reality-Vision" {
		t.Errorf("proxy = %v", p)
	}
	if p["flow"] != "xtls-rprx-vision" {
		t.Errorf("flow = %v", p["flow"])
	}
	if len(cfg.ProxyGroups) != 1 || cfg.ProxyGroups[0]["name"] != "节点选择" {
		t.Errorf("proxy-groups = %v", cfg.ProxyGroups)
	}
	if len(cfg.Rules) == 0 || !strings.HasPrefix(cfg.Rules[0], "MATCH,") {
		t.Errorf("rules = %v", cfg.Rules)
	}
	t.Logf("converted config:\n%s", out)
}

func TestConvertBase64Subscription(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(vlessLink + "\n"))
	out, err := Prepare([]byte(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "type: vless") {
		t.Errorf("base64 订阅未转换：\n%s", out)
	}
}

func TestPreparePassesYAMLThrough(t *testing.T) {
	in := []byte("proxies:\n  - {name: a, type: socks5, server: 127.0.0.1, port: 1080}\nrules:\n  - MATCH,DIRECT\n")
	out, err := Prepare(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(in) {
		t.Errorf("YAML 被改动了：\n%s", out)
	}
}
