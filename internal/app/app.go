// Package app is the application's interface: the window's sidebar, its
// pages, and the state behind them.
package app

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"gopkg.in/yaml.v3"

	"github.com/victorzhao/clash-go/internal/core"
	"github.com/victorzhao/clash-go/internal/mihomoapi"
	"github.com/victorzhao/clash-go/internal/platform"
	"github.com/victorzhao/clash-go/internal/profile"
	"github.com/victorzhao/clash-go/internal/store"
)

// sample is one second of traffic.
type sample struct {
	up, down int64
}

// App is the application's state.
type App struct {
	win   *mygo.Window
	tray  *mygo.Tray
	st    *store.Store
	core  *core.Manager
	api   *mihomoapi.Client
	route *ui.Router

	settings store.Settings
	profiles []store.Profile

	running atomic.Bool
	errMu   sync.Mutex
	coreErr string
	version string

	toastText string

	// traffic is the last minute of the core's traffic, and the totals.
	history            []sample
	up, down           int64
	totalUp, totalDown int64
	memory             uint64
	connCount          int

	// proxies are the core's, with the groups shown and the one chosen.
	proxies map[string]*mihomoapi.Proxy
	groups  []string
	groupAt int
	testing bool

	conns      []mihomoapi.Connection
	connFilter string
	connList   ui.ListState

	rules       []mihomoapi.Rule
	ruleList    ui.ListState
	ruleEditAt  int // -1: adding, >=0: editing that index
	ruleOpen    bool
	ruleType    string
	rulePayload string
	ruleProxy   string

	logs     []core.LogEntry
	logList  ui.ListState
	logPause bool

	// The dashboard: when the core started, the machine's address on the
	// network, the app's own memory, the API's answer time, the connections
	// that ended, and the dashboard's lists.
	startedAt  time.Time
	lanIP      string
	appMem     uint64
	apiLatency int
	// testOK/testDelay are the last check of the test website through the
	// selected proxy; siteOpen/siteDraft drive its edit dialog.
	testOK, testingSite bool
	testDelay           int
	// nodeOK/nodeDelay are the last test of the node the profile selects,
	// which the site test needs first; nodeTested says it ran.
	nodeOK, nodeTested bool
	nodeDelay          int
	siteOpen           bool
	siteDraft          string
	recent             []closedRequest
	dashTab            int
	dashConn           ui.ListState
	dashList           ui.ListState
	// servers maps a proxy's name to its server address, for the profile
	// serversFor; it is read from the profile when the status card asks.
	servers    map[string]string
	serversFor string

	// The profile shown on the configuration page, and its editor.
	viewID     string
	viewText   string
	viewErr    string
	viewEditor *editor.State

	// collapsed hides the sidebar's labels.
	collapsed bool

	// The add-profile dialog.
	addOpen bool
	addTab  int
	addName string
	addURL  string
	addText string
	adding  bool

	// Drafts of the settings' numbers.
	portDraft    float64
	apiPortDraft float64

	// mode is the index of the mode shown in the segmented control.
	mode int
}

// New returns the app with the settings and profiles on disk.
func New() (*App, error) {
	st, err := store.Open()
	if err != nil {
		return nil, err
	}
	return newWith(st)
}

// NewAt returns the app whose files are in dir, for tests.
func NewAt(dir string) (*App, error) {
	st, err := store.OpenAt(dir)
	if err != nil {
		return nil, err
	}
	return newWith(st)
}

func newWith(st *store.Store) (*App, error) {
	settings, profiles, err := st.Load()
	if err != nil {
		return nil, err
	}
	a := &App{
		st:           st,
		route:        ui.NewRouter("/dashboard"),
		settings:     settings,
		profiles:     profiles,
		portDraft:    float64(settings.MixedPort),
		apiPortDraft: float64(settings.APIPort),
	}
	a.mode = modeIndex(settings.Mode)
	a.lanIP = localIP()
	a.core = core.New(st.Dir())
	a.api = mihomoapi.New(fmt.Sprintf("127.0.0.1:%d", settings.APIPort), settings.Secret)
	a.core.OnLog(func(e core.LogEntry) {
		if a.win == nil {
			return
		}
		a.update(func() {
			if a.logPause {
				return
			}
			a.logs = append(a.logs, e)
			if len(a.logs) > 800 {
				a.logs = a.logs[len(a.logs)-800:]
			}
		})
	})
	return a, nil
}

func modeIndex(mode string) int {
	switch mode {
	case "global":
		return 1
	case "direct":
		return 2
	}
	return 0
}

func modeName(i int) string {
	switch i {
	case 1:
		return "global"
	case 2:
		return "direct"
	}
	return "rule"
}

// Attach gives the app its window, and starts the core and the refresher
// when the settings say so.
func (a *App) Attach(win *mygo.Window) {
	a.win = win
	if a.settings.StartCore {
		a.startCore()
	}
	go a.refreshLoop()
	go a.updateLoop()
}

// updateLoop updates subscriptions whose update interval has passed, every
// half hour.
func (a *App) updateLoop() {
	tick := time.NewTicker(30 * time.Minute)
	defer tick.Stop()
	for range tick.C {
		var due []store.Profile
		a.update(func() {
			for _, p := range a.profiles {
				if p.Type != "remote" {
					continue
				}
				hours := p.Interval
				if hours <= 0 {
					hours = a.settings.UpdateInterval
				}
				if hours <= 0 {
					continue
				}
				if time.Since(p.UpdatedAt) >= time.Duration(hours)*time.Hour {
					due = append(due, p)
				}
			}
		})
		for _, p := range due {
			a.updateProfile(p)
		}
	}
}

// update runs fn on the main thread and draws a frame after it, as the
// work of another goroutine does; without a window, as in tests, it runs
// fn there and then.
func (a *App) update(fn func()) {
	if a.win == nil {
		fn()
		return
	}
	a.win.Update(fn)
}

// View builds the window's interface.
func (a *App) View(c *ui.Context) {
	a.applyTheme(c)
	t := c.Theme()
	if a.toastText != "" {
		c.Toast(a.toastText)
		a.toastText = ""
	}
	ui.Row(c).Fill().Children(func() {
		a.sidebar(c)
		ui.Column(c).Grow(1).FillHeight().Background(t.Background).Children(func() {
			a.route.View(c, func(r *ui.Route) {
				switch {
				case r.Match("/proxies"):
					a.proxiesPage(c)
				case r.Match("/profiles/{id}"):
					a.profileViewPage(c, r.Param("id"))
				case r.Match("/profiles"):
					a.profilesPage(c)
				case r.Match("/connections"):
					a.connectionsPage(c)
				case r.Match("/rules"), r.Match("/tools/rules"):
					a.rulesPage(c)
				case r.Match("/logs"), r.Match("/tools/logs"):
					a.logsPage(c)
				case r.Match("/settings"), r.Match("/tools/settings"):
					a.settingsPage(c)
				default:
					a.dashboardPage(c)
				}
			})
		})
	})
}

func (a *App) sidebar(c *ui.Context) {
	t := c.Theme()
	width := float32(210)
	if a.collapsed {
		width = 64
	}
	ui.Column(c).Width(width).FillHeight().Padding(12, 10).Gap(6).
		Background(sidebarColor(c)).Children(func() {
		// The collapse toggle sits alone at the top, as in FlClash.
		toggle := ui.ButtonBase(c.Key("sidebar-toggle")).Size(34, 34).Radius(8).Center()
		toggle.Label(map[bool]string{true: "展开侧栏", false: "收起侧栏"}[a.collapsed]).
			Tooltip(map[bool]string{true: "展开侧栏", false: "收起侧栏"}[a.collapsed])
		toggle.Children(func() {
			ui.Icon(c, iconPanel).FontSize(17).TextColor(t.TextMuted)
		})
		if toggle.Clicked() {
			a.collapsed = !a.collapsed
		}
		ui.Box(c).Height(6)
		a.nav(c, "/dashboard", "仪表盘", iconDashboard)
		a.nav(c, "/proxies", "代理", iconProxies)
		a.navGroup(c, "/profiles", "配置", iconProfiles)
		a.nav(c, "/connections", "请求", iconConnections)
		a.nav(c, "/rules", "规则", iconRules)
		a.nav(c, "/logs", "日志", iconLogs)
		a.nav(c, "/settings", "设置", iconSettings)
	})
}

func (a *App) nav(c *ui.Context, path, label string, ic *ui.SVG) {
	a.navItem(c, path, label, ic, false)
}

// navGroup is a nav item that stays active on its sub-pages, such as Tools.
func (a *App) navGroup(c *ui.Context, path, label string, ic *ui.SVG) {
	a.navItem(c, path, label, ic, true)
}

func (a *App) navItem(c *ui.Context, path, label string, ic *ui.SVG, group bool) {
	t := c.Theme()
	active := a.route.Path() == path
	if group && strings.HasPrefix(a.route.Path(), path+"/") {
		active = true
	}
	item := ui.ButtonBase(c.Key("nav" + path)).Height(40).Radius(10).AlignItems(ui.Center).Justify(ui.Start).
		Transition(ui.ElementTransition{Colors: true})
	if a.collapsed {
		item.Center().Tooltip(label).Label(label)
	} else {
		item.PaddingX(12).Gap(14)
	}
	hovered := item.Hovered()
	switch {
	case active:
		item.Background(navActiveColor(c)).TextColor(t.Accent)
	case hovered:
		item.Background(pillColor(c)).TextColor(t.Text)
	default:
		item.TextColor(t.TextMuted)
	}
	item.Children(func() {
		ui.Icon(c, ic).FontSize(17)
		if !a.collapsed {
			ui.Text(c, label).FontSize(15)
		}
	})
	if item.Clicked() {
		a.route.Push(path)
	}
}

// subPageHeader heads a page reached from another one, with the button
// that goes back.
func (a *App) subPageHeader(c *ui.Context, title, subtitle string) {
	t := c.Theme()
	ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
		ui.BackButton(c, a.route).Label("返回")
		ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, title).FontSize(22).Bold().SingleLine()
			if subtitle != "" {
				ui.Text(c, subtitle).FontSize(12).TextColor(t.TextMuted).SingleLine()
			}
		})
	})
}

// pageTitle heads a page.
func (a *App) pageTitle(c *ui.Context, title, subtitle string) {
	t := c.Theme()
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, title).FontSize(22).Bold()
		if subtitle != "" {
			ui.Text(c, subtitle).FontSize(12).TextColor(t.TextMuted)
		}
	})
}

func (a *App) coreStatus() (bool, string) {
	a.errMu.Lock()
	defer a.errMu.Unlock()
	return a.running.Load(), a.coreErr
}

func (a *App) setErr(err string) {
	a.errMu.Lock()
	a.coreErr = err
	a.errMu.Unlock()
}

// startCore applies the active profile with the settings and starts the
// core.
func (a *App) startCore() {
	if a.running.Load() {
		return
	}
	p, ok := a.activeProfile()
	if !ok {
		if len(a.profiles) == 0 {
			a.toastText = "请先在「配置」中导入一个配置"
			return
		}
		p = a.profiles[0]
		a.settings.ActiveProfile = p.ID
		_ = a.st.SaveSettings(a.settings)
	}
	content, err := a.st.ProfileContent(p.ID)
	if err != nil {
		a.toastText = "读取配置失败：" + err.Error()
		return
	}
	s := a.settings
	go func() {
		err := a.core.Start(content, s)
		if err == nil {
			a.api = mihomoapi.New(fmt.Sprintf("127.0.0.1:%d", s.APIPort), s.Secret)
			if s.SystemProxy {
				platform.SetSystemProxy(true, fmt.Sprintf("127.0.0.1:%d", s.MixedPort))
			}
		}
		a.update(func() {
			a.running.Store(err == nil)
			if err != nil {
				a.setErr(err.Error())
				a.toastText = "启动失败：" + err.Error()
			} else {
				a.setErr("")
				a.startedAt = time.Now()
				a.toastText = "内核已启动"
				a.refreshTray()
				a.testSite()
			}
		})
	}()
}

// stopCore stops the core and turns the system proxy off.
func (a *App) stopCore() {
	if !a.running.Load() {
		return
	}
	go func() {
		a.core.Stop()
		platform.SetSystemProxy(false, "")
		a.update(func() {
			a.running.Store(false)
			a.startedAt = time.Time{}
			a.toastText = "内核已停止"
			a.refreshTray()
			a.history = nil
			a.up, a.down = 0, 0
		})
	}()
}

// restartCore applies the settings again, for the ones that take a reload.
func (a *App) restartCore() {
	if !a.running.Load() {
		return
	}
	go func() {
		a.core.Stop()
		p, ok := a.activeProfile()
		if !ok {
			a.update(func() { a.running.Store(false) })
			return
		}
		content, err := a.st.ProfileContent(p.ID)
		if err != nil {
			a.update(func() { a.running.Store(false); a.setErr(err.Error()) })
			return
		}
		s := a.settings
		err = a.core.Start(content, s)
		a.update(func() {
			a.running.Store(err == nil)
			if err != nil {
				a.setErr(err.Error())
			} else {
				a.setErr("")
				a.startedAt = time.Now()
				a.testSite()
			}
		})
	}()
}

func (a *App) activeProfile() (store.Profile, bool) {
	for _, p := range a.profiles {
		if p.ID == a.settings.ActiveProfile {
			return p, true
		}
	}
	return store.Profile{}, false
}

// findProfile returns the profile with id.
func (a *App) findProfile(id string) (store.Profile, bool) {
	for _, p := range a.profiles {
		if p.ID == id {
			return p, true
		}
	}
	return store.Profile{}, false
}

// saveSettings writes the settings and applies the changes that need the
// core restarted.
func (a *App) saveSettings(restart bool) {
	a.settings.MixedPort = int(a.portDraft)
	a.settings.APIPort = int(a.apiPortDraft)
	a.settings.Mode = modeName(a.mode)
	if err := a.st.SaveSettings(a.settings); err != nil {
		a.toastText = "保存设置失败：" + err.Error()
		return
	}
	a.api = mihomoapi.New(fmt.Sprintf("127.0.0.1:%d", a.settings.APIPort), a.settings.Secret)
	if exe, err := os.Executable(); err == nil {
		platform.SetAutoStart(a.settings.AutoStart, exe)
	}
	if restart {
		a.restartCore()
	}
}

// setMode changes the core's mode from the tray or a page.
func (a *App) setMode(mode string) {
	a.mode = modeIndex(mode)
	a.settings.Mode = mode
	if err := a.st.SaveSettings(a.settings); err != nil {
		a.toastText = "保存设置失败：" + err.Error()
		return
	}
	a.restartCore()
	a.refreshTray()
}

func (a *App) refreshLoop() {
	trafficTick := time.NewTicker(time.Second)
	slowTick := time.NewTicker(2 * time.Second)
	siteTick := time.NewTicker(30 * time.Second)
	defer trafficTick.Stop()
	defer slowTick.Stop()
	defer siteTick.Stop()
	for {
		select {
		case <-siteTick.C:
			if a.running.Load() {
				a.update(func() { a.testSite() })
			}
		case <-trafficTick.C:
			if !a.running.Load() {
				continue
			}
			t, err := a.api.Traffic()
			if err != nil {
				continue
			}
			a.update(func() {
				a.up, a.down = t.Up, t.Down
				a.totalUp += t.Up
				a.totalDown += t.Down
				a.history = append(a.history, sample{up: t.Up, down: t.Down})
				if len(a.history) > 60 {
					a.history = a.history[len(a.history)-60:]
				}
			})
		case <-slowTick.C:
			if !a.running.Load() {
				continue
			}
			conns, err := a.api.Connections()
			if err != nil {
				continue
			}
			proxies, perr := a.api.Proxies()
			var rules []mihomoapi.Rule
			if r, rerr := a.api.Rules(); rerr == nil {
				rules = r
			}
			asked := time.Now()
			version, verr := a.api.Version()
			latency := int(time.Since(asked) / time.Millisecond)
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			a.update(func() {
				a.noteClosed(conns.Connections)
				a.conns = conns.Connections
				a.connCount = len(conns.Connections)
				a.memory = conns.Memory
				a.appMem = ms.Alloc
				a.apiLatency = latency
				if perr == nil {
					a.setProxies(proxies)
				}
				if rules != nil {
					a.rules = rules
				}
				if verr == nil {
					a.version = version
				}
			})
		}
	}
}

// setProxies keeps the groups of the core's proxies, keeping the group
// chosen when it is still there.
func (a *App) setProxies(proxies map[string]*mihomoapi.Proxy) {
	prev := ""
	if a.groupAt >= 0 && a.groupAt < len(a.groups) {
		prev = a.groups[a.groupAt]
	}
	var groups []string
	for name, p := range proxies {
		if len(p.All) > 0 {
			groups = append(groups, name)
		}
	}
	sort.Strings(groups)
	a.proxies = proxies
	a.groups = groups
	a.groupAt = 0
	for i, g := range groups {
		if g == prev {
			a.groupAt = i
			break
		}
	}
}

// group returns the proxy group shown on the proxies page.
func (a *App) group() *mihomoapi.Proxy {
	if a.groupAt < 0 || a.groupAt >= len(a.groups) {
		return nil
	}
	return a.proxies[a.groups[a.groupAt]]
}

// selectProxy makes the group shown use name.
func (a *App) selectProxy(name string) {
	g := a.group()
	if g == nil {
		return
	}
	group := g.Name
	go func() {
		err := a.api.Select(group, name)
		a.update(func() {
			if err != nil {
				a.toastText = "切换失败：" + err.Error()
				return
			}
			if p := a.proxies[group]; p != nil {
				p.Now = name
			}
			a.refreshTray()
		})
	}()
}

// testGroup tests the delays of the proxies of the group shown.
func (a *App) testGroup() {
	g := a.group()
	if g == nil || a.testing {
		return
	}
	group := g.Name
	a.testing = true
	go func() {
		// The node check uses an IP literal: a delay test resolves the URL's
		// host locally first, and a poisoned answer would fail it.
		results, err := a.api.GroupDelay(group, nodeCheckURL, 5*time.Second)
		a.update(func() {
			a.testing = false
			if err != nil {
				a.toastText = "测速失败：" + err.Error()
				return
			}
			for name, delay := range results {
				if p := a.proxies[name]; p != nil {
					p.Delay = delay
				}
			}
		})
	}()
}

// testSite checks the configured test website through the proxy currently
// selected, and records the result for the dashboard's service card. It
// tests the node first, and only a node that connects gets the site test:
// the site goes through the core's own proxy port, as a browser does, so it
// follows the profile's rules and the node resolves the name itself, which
// a local DNS answer (poisoned, as binance.com's often is) cannot spoil.
func (a *App) testSite() {
	if a.testingSite {
		return
	}
	if !a.running.Load() {
		a.toastText = "内核未运行，无法检测网站"
		return
	}
	proxy := "DIRECT"
	if g := a.group(); g != nil && g.Now != "" {
		proxy = g.Now
	}
	site := strings.TrimSpace(a.settings.TestURL)
	if site == "" {
		site = store.Default().TestURL
	}
	mixedPort := a.settings.MixedPort
	a.testingSite = true
	go func() {
		// The node first: a site test means nothing while the proxy is not
		// connected, so check the node against an address that needs no DNS,
		// and only then take the site through the core.
		delay, err := a.api.Delay(proxy, nodeCheckURL, 5*time.Second)
		nodeOK := err == nil && delay > 0
		siteDelay := 0
		siteOK := false
		if nodeOK {
			siteDelay, err = siteReachable(site, mixedPort)
			siteOK = err == nil
		}
		a.update(func() {
			a.testingSite = false
			a.nodeTested = true
			a.nodeOK = nodeOK
			a.nodeDelay = delay
			a.testOK = nodeOK && siteOK
			a.testDelay = siteDelay
			if !nodeOK {
				a.toastText = "节点未连接，跳过网站测试"
			}
		})
	}()
}

// nodeCheckURL is an address the node test connects to before the site test.
// It is an IP literal, so no DNS answer can make the node look broken.
const nodeCheckURL = "http://223.5.5.5"

// siteReachable requests site through the core's proxy port, as a browser
// does, and returns how long the request took in milliseconds. Any HTTP
// answer counts: a site that answers 403 is reachable.
func siteReachable(site string, port int) (int, error) {
	proxy, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	if err != nil {
		return 0, err
	}
	client := &http.Client{
		Timeout: 12 * time.Second,
		Transport: &http.Transport{
			Proxy:                 http.ProxyURL(proxy),
			TLSHandshakeTimeout:   8 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}
	start := time.Now()
	resp, err := client.Get(site)
	if err != nil {
		return 0, err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	ms := int(time.Since(start) / time.Millisecond)
	if ms == 0 {
		ms = 1
	}
	return ms, nil
}

// nodeServer returns where the proxy named name connects, as the profile
// gives it ("43.167.27.194:443"), or "" for a node of a provider, which the
// profile's file does not hold.
func (a *App) nodeServer(name string) string {
	p, ok := a.activeProfile()
	if !ok {
		return ""
	}
	if a.serversFor != p.ID {
		a.serversFor = p.ID
		a.servers = nil
		if content, err := a.st.ProfileContent(p.ID); err == nil {
			var cfg struct {
				Proxies []struct {
					Name   string `yaml:"name"`
					Server string `yaml:"server"`
					Port   any    `yaml:"port"`
				} `yaml:"proxies"`
			}
			if yaml.Unmarshal(content, &cfg) == nil {
				servers := make(map[string]string, len(cfg.Proxies))
				for _, proxy := range cfg.Proxies {
					if proxy.Server == "" {
						continue
					}
					address := proxy.Server
					if proxy.Port != nil {
						address += ":" + fmt.Sprint(proxy.Port)
					}
					servers[proxy.Name] = address
				}
				a.servers = servers
			}
		}
	}
	return a.servers[name]
}

// nodeStatus returns the node the profile selects, its last delay, and
// whether it connects.
func (a *App) nodeStatus() (string, int, bool) {
	g := a.group()
	if g == nil || g.Now == "" {
		return "", 0, false
	}
	delay := 0
	if p := a.proxies[g.Now]; p != nil {
		delay = p.Delay
	}
	if a.nodeTested {
		delay = a.nodeDelay
	}
	return g.Now, delay, delay > 0
}

// trayMenu builds the tray's menu, which MyGo shows on a right click, as
// the app's other menus.
func (a *App) trayMenu() *mygo.Menu {
	check := func(mode string) bool { return a.settings.Mode == mode }
	core := &mygo.MenuItem{Label: "启动内核", Click: func(*mygo.MenuItem, *mygo.Window) { a.startCore() }}
	if a.running.Load() {
		core = &mygo.MenuItem{Label: "停止内核", Click: func(*mygo.MenuItem, *mygo.Window) { a.stopCore() }}
	}
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: "显示主窗口", Click: func(*mygo.MenuItem, *mygo.Window) { a.showWindow() }},
		mygo.Separator(),
		{Label: "模式", Submenu: []*mygo.MenuItem{
			{Label: "规则", Type: mygo.MenuItemRadio, Checked: check("rule"), Click: func(*mygo.MenuItem, *mygo.Window) { a.setMode("rule") }},
			{Label: "全局", Type: mygo.MenuItemRadio, Checked: check("global"), Click: func(*mygo.MenuItem, *mygo.Window) { a.setMode("global") }},
			{Label: "直连", Type: mygo.MenuItemRadio, Checked: check("direct"), Click: func(*mygo.MenuItem, *mygo.Window) { a.setMode("direct") }},
		}},
		mygo.Separator(),
		core,
		mygo.Separator(),
		{Role: mygo.RoleQuit},
	})
}

// refreshTray gives the tray its menu again, for the state it shows.
func (a *App) refreshTray() {
	if a.tray != nil {
		a.tray.SetMenu(a.trayMenu())
	}
}

// SetTray keeps the tray icon the app created and wires its right-click to
// open the menu above the icon.
func (a *App) SetTray(t *mygo.Tray) {
	a.tray = t
	a.refreshTray()
}

// showWindow brings the main window to the front.
func (a *App) showWindow() {
	if a.win != nil {
		a.win.Show()
		a.win.Focus()
	}
}

// menuOwner returns a window handle for the native popup menu to use as its
// owner, so the menu closes when the user clicks elsewhere.
func (a *App) menuOwner() uintptr {
	if a.win == nil {
		return 0
	}
	return a.win.NativeHandle()
}

// setSystemProxy points the system's proxy at the mixed port, or turns it
// off.
func setSystemProxy(on bool, port int) {
	if on {
		platform.SetSystemProxy(true, fmt.Sprintf("127.0.0.1:%d", port))
		return
	}
	platform.SetSystemProxy(false, "")
}

// Shutdown stops the core and turns the system proxy off, as the app
// quits.
func (a *App) Shutdown() {
	a.core.Stop()
	platform.SetSystemProxy(false, "")
}

// addProfile saves a downloaded or pasted configuration as a profile.
func (a *App) addProfile(name, kind, u string, content []byte, info profile.Info) {
	id := store.NewProfileID()
	if err := a.st.WriteProfile(id, content); err != nil {
		a.toastText = "保存配置失败：" + err.Error()
		return
	}
	p := store.Profile{ID: id, Name: name, Type: kind, URL: u, UpdatedAt: time.Now(),
		Used: info.Upload + info.Download, Total: info.Total, Expire: info.Expire}
	a.profiles = append(a.profiles, p)
	a.st.SaveProfiles(a.profiles)
	if a.settings.ActiveProfile == "" {
		a.settings.ActiveProfile = id
		a.st.SaveSettings(a.settings)
	}
	a.toastText = "已导入配置「" + name + "」"
}

// saveRules writes the rules back to the active profile's YAML, keeping all
// other sections intact, and restarts the core so the change takes effect.
func (a *App) saveRules(rules []mihomoapi.Rule) {
	p, ok := a.activeProfile()
	if !ok {
		a.toastText = "没有正在使用的配置"
		return
	}
	content, err := a.st.ProfileContent(p.ID)
	if err != nil {
		a.toastText = "读取配置失败：" + err.Error()
		return
	}
	lines := make([]string, 0, len(rules))
	for _, r := range rules {
		// MATCH takes no payload: "MATCH,DIRECT", not "MATCH,,DIRECT".
		line := ruleTypeConfig(r.Type)
		if strings.TrimSpace(r.Payload) != "" {
			line += "," + r.Payload
		}
		lines = append(lines, line+","+r.Proxy)
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		a.toastText = "解析配置失败：" + err.Error()
		return
	}
	cfg["rules"] = lines
	out, err := yaml.Marshal(cfg)
	if err != nil {
		a.toastText = "保存规则失败：" + err.Error()
		return
	}
	if err := a.st.WriteProfile(p.ID, out); err != nil {
		a.toastText = "保存规则失败：" + err.Error()
		return
	}
	a.rules = rules
	if a.running.Load() {
		a.restartCore()
		a.toastText = "规则已保存并重启内核"
	} else {
		a.toastText = "规则已保存，可从仪表盘启动内核"
	}
}

// loadRules reads the rules of the active profile from disk. It is used
// when the core is not running (for example after a bad rule prevented it
// from starting), so the rules can still be inspected and fixed.
func (a *App) loadRules() {
	p, ok := a.activeProfile()
	if !ok {
		a.rules = nil
		return
	}
	content, err := a.st.ProfileContent(p.ID)
	if err != nil {
		return
	}
	var cfg struct {
		Rules []string `yaml:"rules"`
	}
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return
	}
	rules := make([]mihomoapi.Rule, 0, len(cfg.Rules))
	for _, line := range cfg.Rules {
		parts := strings.Split(line, ",")
		r := mihomoapi.Rule{}
		if len(parts) >= 3 {
			r.Type, r.Payload, r.Proxy = parts[0], parts[1], parts[2]
		} else if len(parts) == 2 {
			// MATCH has no payload: "MATCH,proxy".
			r.Type, r.Proxy = parts[0], parts[1]
		}
		if r.Type != "" {
			rules = append(rules, r)
		}
	}
	a.rules = rules
}

// updateProfile downloads a remote profile again.
func (a *App) updateProfile(p store.Profile) {
	if p.URL == "" {
		a.toastText = "本地配置无需更新"
		return
	}
	go func() {
		data, info, err := profile.Fetch(p.URL)
		a.update(func() {
			if err != nil {
				a.toastText = "更新失败：" + err.Error()
				return
			}
			if err := a.st.WriteProfile(p.ID, data); err != nil {
				a.toastText = "保存失败：" + err.Error()
				return
			}
			for i := range a.profiles {
				if a.profiles[i].ID == p.ID {
					a.profiles[i].UpdatedAt = time.Now()
					if info.Interval > 0 {
						a.profiles[i].Interval = info.Interval
					}
					a.profiles[i].Used = info.Upload + info.Download
					a.profiles[i].Total = info.Total
					a.profiles[i].Expire = info.Expire
				}
			}
			a.st.SaveProfiles(a.profiles)
			if a.viewID == p.ID {
				a.viewID = ""
			}
			a.toastText = "已更新「" + p.Name + "」"
			if a.settings.ActiveProfile == p.ID {
				a.restartCore()
			}
		})
	}()
}

// removeProfile deletes a profile and its file.
func (a *App) removeProfile(p store.Profile) {
	_ = a.st.DeleteProfile(p.ID)
	for i := range a.profiles {
		if a.profiles[i].ID == p.ID {
			a.profiles = append(a.profiles[:i], a.profiles[i+1:]...)
			break
		}
	}
	_ = a.st.SaveProfiles(a.profiles)
	if a.settings.ActiveProfile == p.ID {
		a.settings.ActiveProfile = ""
		if len(a.profiles) > 0 {
			a.settings.ActiveProfile = a.profiles[0].ID
		}
		_ = a.st.SaveSettings(a.settings)
		a.restartCore()
	}
	a.toastText = "已删除「" + p.Name + "」"
}

// closedRequest is a connection that ended, shown on the dashboard.
type closedRequest struct {
	Host   string
	Chains []string
	Down   int64
	Up     int64
	At     time.Time
}

// noteClosed keeps the connections that ended since the last frame, for
// the dashboard's recent requests.
func (a *App) noteClosed(current []mihomoapi.Connection) {
	now := make(map[string]bool, len(current))
	for _, c := range current {
		now[c.ID] = true
	}
	for _, c := range a.conns {
		if now[c.ID] {
			continue
		}
		host := c.Metadata.Host
		if host == "" {
			host = c.Metadata.Destination
		}
		a.recent = append([]closedRequest{{Host: host, Chains: c.Chains, Down: c.Download, Up: c.Upload, At: time.Now()}}, a.recent...)
	}
	if len(a.recent) > 50 {
		a.recent = a.recent[:50]
	}
}

// localIP returns the machine's address on the local network, or "—".
func localIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "—"
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
				continue
			}
			return ip4.String()
		}
	}
	return "—"
}

// defaultPolicy returns the strategy of the MATCH rule, the one that takes
// the connections no rule matches, and whether the profile has one.
func (a *App) defaultPolicy() (string, bool) {
	for i := len(a.rules) - 1; i >= 0; i-- {
		if ruleTypeConfig(a.rules[i].Type) == "MATCH" {
			return a.rules[i].Proxy, true
		}
	}
	return "", false
}

// defaultPolicies lists the strategies the default policy can be set to.
func (a *App) defaultPolicies() []string {
	out := []string{"DIRECT", "REJECT"}
	seen := map[string]bool{"DIRECT": true, "REJECT": true}
	for _, name := range a.groups {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	if current, ok := a.defaultPolicy(); ok && !seen[current] {
		out = append(out, current)
	}
	return out
}

// setDefaultPolicy makes the MATCH rule point at proxy, adding the rule when
// the profile has none.
func (a *App) setDefaultPolicy(proxy string) {
	proxy = strings.TrimSpace(proxy)
	if proxy == "" {
		return
	}
	next := append([]mihomoapi.Rule(nil), a.rules...)
	at := -1
	for i := len(next) - 1; i >= 0; i-- {
		if ruleTypeConfig(next[i].Type) == "MATCH" {
			at = i
			break
		}
	}
	rule := mihomoapi.Rule{Type: "MATCH", Proxy: proxy}
	if at >= 0 {
		next[at] = rule
	} else {
		next = append(next, rule)
	}
	a.saveRules(next)
}
