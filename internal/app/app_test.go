package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/victorzhao/clash-go/internal/mihomoapi"
	"github.com/victorzhao/clash-go/internal/profile"
	"github.com/victorzhao/clash-go/internal/store"
)

// The view builds every page without a window.
func TestPagesBuild(t *testing.T) {
	a, err := NewAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.View, 1080, 720)
	tt.Frame()

	pages := []struct {
		nav  string
		want string
	}{
		{"代理", "内核未运行，启动后才能查看和切换代理。"},
		{"配置", "还没有配置"},
		{"请求", "内核未运行，启动后才能看到请求。"},
		{"规则", "内核未运行 · 显示配置文件中的规则"},
		{"日志", "还没有日志"},
		{"设置", "修改后立即保存；需要重载的设置会重启内核"},
	}
	for _, p := range pages {
		if err := tt.Click(p.nav); err != nil {
			t.Fatalf("点击 %q: %v", p.nav, err)
		}
		if !tt.HasText(p.want) {
			t.Errorf("点击 %q 后没有 %q，文本：%q", p.nav, p.want, tt.Texts())
		}
	}
	if err := tt.Click("仪表盘"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("出站模式") {
		t.Errorf("仪表盘没有出站模式，文本：%q", tt.Texts())
	}
}

// The add-profile dialog opens and closes.
func TestAddProfileDialog(t *testing.T) {
	a, err := NewAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.View, 1080, 720)
	tt.Frame()
	if err := tt.Click("配置"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("添加配置"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("订阅链接") || !tt.HasText("手动输入") {
		t.Fatalf("对话框没有标签页，文本：%q", tt.Texts())
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	if a.addOpen {
		t.Error("取消后对话框还开着")
	}
}

// saveRules rewrites the active profile's rules and keeps other sections.
func TestSaveRules(t *testing.T) {
	a, err := NewAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := store.Profile{ID: "p1", Name: "test", Type: "local"}
	a.profiles = []store.Profile{p}
	a.settings.ActiveProfile = p.ID
	initial := "port: 7890\nproxies:\n  - {name: a, type: socks5, server: 127.0.0.1, port: 1080}\nrules:\n  - MATCH,DIRECT\n"
	if err := a.st.WriteProfile(p.ID, []byte(initial)); err != nil {
		t.Fatal(err)
	}
	rules := []mihomoapi.Rule{{Type: "DOMAIN", Payload: "example.com", Proxy: "a"}}
	a.saveRules(rules)
	data, err := a.st.ProfileContent(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "DOMAIN,example.com,a") {
		t.Errorf("新规则没写入：\n%s", s)
	}
	if !strings.Contains(s, "socks5") || !strings.Contains(s, "port: 7890") {
		t.Errorf("其他配置被破坏：\n%s", s)
	}
	if len(a.rules) != 1 || a.rules[0].Payload != "example.com" {
		t.Errorf("内存中的规则未更新：%v", a.rules)
	}
}

func TestLoadRulesWhileStopped(t *testing.T) {
	a, err := NewAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := store.Profile{ID: "p1", Name: "test", Type: "local"}
	a.profiles = []store.Profile{p}
	a.settings.ActiveProfile = p.ID
	initial := "proxies: []\nrules:\n  - DOMAIN-SUFFIX,example.com,DIRECT\n  - MATCH,proxy\n"
	if err := a.st.WriteProfile(p.ID, []byte(initial)); err != nil {
		t.Fatal(err)
	}
	a.loadRules()
	if len(a.rules) != 2 {
		t.Fatalf("rules = %v, want 2", a.rules)
	}
	if a.rules[0].Type != "DOMAIN-SUFFIX" || a.rules[0].Payload != "example.com" || a.rules[0].Proxy != "DIRECT" {
		t.Errorf("rule 0 = %v", a.rules[0])
	}
	if a.rules[1].Type != "MATCH" || a.rules[1].Proxy != "proxy" {
		t.Errorf("rule 1 = %v", a.rules[1])
	}
}

func TestRuleTypeConfig(t *testing.T) {
	cases := map[string]string{
		"DomainSuffix":   "DOMAIN-SUFFIX",
		"RuleSet":        "RULE-SET",
		"IPCIDR":         "IP-CIDR",
		"MATCH":          "MATCH",
		"DOMAIN-KEYWORD": "DOMAIN-KEYWORD",
	}
	for in, want := range cases {
		if got := ruleTypeConfig(in); got != want {
			t.Errorf("ruleTypeConfig(%q) = %q, want %q", in, got, want)
		}
	}
}

// The configuration page shows a profile's YAML, read-only.
func TestProfileView(t *testing.T) {
	a, err := NewAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.addProfile("测试配置", "local", "", []byte(`proxies:
  - {name: a, type: socks5, server: 127.0.0.1, port: 1080}
rules:
  - MATCH,DIRECT
`), profile.Info{})
	tt := ui.NewTester(a.View, 1200, 780)
	tt.Frame()
	if err := tt.Click("配置"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("查看"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second) // the router's page transition
	if a.viewID == "" {
		t.Fatal("没有打开配置页")
	}
	if a.viewEditor == nil {
		t.Fatal("没有建立编辑器")
	}
	if !tt.HasText("测试配置") {
		t.Errorf("配置页没有名称，文本：%q", tt.Texts())
	}
	if got := a.viewEditor.Buffer.String(); !strings.Contains(got, "socks5") {
		t.Errorf("编辑器内容 %q 没有配置", got)
	}
	if err := tt.Click("返回"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if !strings.HasPrefix(a.route.Path(), "/profiles") || strings.Count(a.route.Path(), "/") != 1 {
		t.Errorf("返回后路径 %q", a.route.Path())
	}
}

// The rule editor offers the built-in strategies and the profile's groups,
// not "proxy" or the nodes themselves.
func TestRuleStrategies(t *testing.T) {
	a, err := NewAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.groups = []string{"节点选择", "自动选择"}
	a.proxies = map[string]*mihomoapi.Proxy{
		"节点选择":  {Name: "节点选择", Type: "Selector", All: []string{"香港 01"}},
		"自动选择":  {Name: "自动选择", Type: "URLTest", All: []string{"香港 01"}},
		"香港 01": {Name: "香港 01", Type: "Shadowsocks"},
	}
	list := a.ruleProxies()
	for _, want := range []string{"DIRECT", "REJECT", "节点选择", "自动选择"} {
		if !slices.Contains(list, want) {
			t.Errorf("策略列表 %q 缺少 %q", list, want)
		}
	}
	for _, unwanted := range []string{"proxy", "香港 01"} {
		if slices.Contains(list, unwanted) {
			t.Errorf("策略列表 %q 不该有 %q", list, unwanted)
		}
	}
	// A rule pointing at "proxy" keeps it while it is edited.
	a.ruleProxy = "proxy"
	if !slices.Contains(a.ruleProxies(), "proxy") {
		t.Error("编辑规则时丢了它自己的策略")
	}
}

// The service card's refresh button tests the website again.
func TestServiceRefresh(t *testing.T) {
	a, err := NewAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"delay":42}`))
	}))
	defer server.Close()
	a.api = mihomoapi.New(strings.TrimPrefix(server.URL, "http://"), "")
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()
	if u, err := url.Parse(proxyServer.URL); err == nil {
		if port, err := strconv.Atoi(u.Port()); err == nil {
			a.settings.MixedPort = port
		}
	}
	a.running.Store(true)
	a.groups = []string{"节点选择"}
	a.proxies = map[string]*mihomoapi.Proxy{"节点选择": {Name: "节点选择", Type: "Selector", Now: "香港 01"}}
	// The fake proxy answers plain HTTP requests; an https site would need
	// it to handle CONNECT and a TLS handshake.
	a.settings.TestURL = "http://www.binance.com/"

	tt := ui.NewTester(a.View, 1200, 780)
	tt.Frame()
	if err := tt.Click("重新检测"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for a.testingSite && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !a.testOK || a.testDelay <= 0 {
		t.Fatalf("检测结果 delay=%d ok=%v", a.testDelay, a.testOK)
	}
	tt.Frame()
	if !tt.HasText("网站可达") || !tt.HasText(fmt.Sprintf("%d ms", a.testDelay)) {
		t.Errorf("卡片没有显示结果，文本：%q", tt.Texts())
	}
}

// The rules page sets the default policy, the MATCH rule that takes the
// connections no rule matches, and saves it in the profile.
func TestDefaultPolicy(t *testing.T) {
	a, err := NewAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.addProfile("测试配置", "local", "", []byte(`proxies:
  - {name: 香港 01, type: socks5, server: 127.0.0.1, port: 1080}
proxy-groups:
  - {name: 节点选择, type: select, proxies: [香港 01]}
rules:
  - DOMAIN-SUFFIX,binance.com,节点选择
`), profile.Info{})
	a.groups = []string{"节点选择"}
	a.proxies = map[string]*mihomoapi.Proxy{"节点选择": {Name: "节点选择", Type: "Selector", Now: "香港 01"}}
	a.rules = []mihomoapi.Rule{{Type: "DomainSuffix", Payload: "binance.com", Proxy: "节点选择"}}

	if _, ok := a.defaultPolicy(); ok {
		t.Error("没有 MATCH 规则时 defaultPolicy 却报有")
	}
	if got := a.defaultPolicies(); len(got) != 3 || got[0] != "DIRECT" || got[1] != "REJECT" || got[2] != "节点选择" {
		t.Errorf("默认策略列表 %q", got)
	}
	a.setDefaultPolicy("节点选择")
	if got, ok := a.defaultPolicy(); !ok || got != "节点选择" {
		t.Errorf("设置后默认策略 %q ok=%v", got, ok)
	}
	// The MATCH rule saves as "MATCH,节点选择", which mihomo reads.
	p, _ := a.activeProfile()
	content, err := a.st.ProfileContent(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "- MATCH,节点选择") {
		t.Errorf(`配置里的规则：
%s`, content)
	}
	// Changing it replaces the rule instead of adding another one.
	a.setDefaultPolicy("REJECT")
	count := 0
	for _, r := range a.rules {
		if ruleTypeConfig(r.Type) == "MATCH" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("MATCH 规则 %d 条", count)
	}
	if got, _ := a.defaultPolicy(); got != "REJECT" {
		t.Errorf("修改后默认策略 %q", got)
	}
}

// The rules page shows the default policy picker.
func TestDefaultPolicyUI(t *testing.T) {
	a, err := NewAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.addProfile("测试配置", "local", "", []byte(`proxies:
  - {name: 香港 01, type: socks5, server: 127.0.0.1, port: 1080}
proxy-groups:
  - {name: 节点选择, type: select, proxies: [香港 01]}
rules:
  - DOMAIN-SUFFIX,binance.com,节点选择
`), profile.Info{})
	a.groups = []string{"节点选择"}
	tt := ui.NewTester(a.View, 1200, 780)
	tt.Frame()
	if err := tt.Click("规则"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if !tt.HasText("默认策略") {
		t.Fatalf("规则页没有默认策略，文本：%q", tt.Texts())
	}
	if !tt.HasText("配置里没有 MATCH 规则") {
		t.Errorf("没有提示缺 MATCH 规则，文本：%q", tt.Texts())
	}
}

// The site test waits for the node: when the node does not connect, the
// site is not tested at all, and the cards say so.
func TestServiceTestNeedsNode(t *testing.T) {
	a, err := NewAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// The API answers the node test with a failure; the proxy port must not
	// be asked for the site at all.
	siteCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"message":"An error occurred in the delay test"}`))
	}))
	defer server.Close()
	a.api = mihomoapi.New(strings.TrimPrefix(server.URL, "http://"), "")
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siteCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()
	if u, err := url.Parse(proxyServer.URL); err == nil {
		if port, err := strconv.Atoi(u.Port()); err == nil {
			a.settings.MixedPort = port
		}
	}
	a.running.Store(true)
	a.groups = []string{"节点选择"}
	a.proxies = map[string]*mihomoapi.Proxy{"节点选择": {Name: "节点选择", Type: "Selector", Now: "香港 01"}}
	// The fake proxy answers plain HTTP requests; an https site would need
	// it to handle CONNECT and a TLS handshake.
	a.settings.TestURL = "http://www.binance.com/"

	tt := ui.NewTester(a.View, 1200, 780)
	tt.Frame()
	if err := tt.Click("重新检测"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for a.testingSite && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if a.nodeOK || a.testOK {
		t.Errorf("节点没连上却报成功: nodeOK=%v testOK=%v", a.nodeOK, a.testOK)
	}
	if siteCalls != 0 {
		t.Errorf("节点未连接时还测了网站 %d 次", siteCalls)
	}
	tt.Frame()
	if !tt.HasText("节点未连接") {
		t.Errorf("卡片没有说明节点未连接，文本：%q", tt.Texts())
	}
}
