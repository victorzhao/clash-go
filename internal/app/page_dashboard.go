package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// card builds a card of the dashboard: a white rounded box with its
// shadow, as FlClash's dashboard lays its cards out.
func card(c *ui.Context) ui.Element {
	return ui.Column(c).Padding(18).Gap(8).Radius(20).Basis(0).MinWidth(0).
		Background(cardColor(c)).Border(1, cardEdge(c)).Shadow(0, 2, 8, 0, cardShadow(c))
}

// cardHead heads a card with an icon and a label.
func cardHead(c *ui.Context, ic *ui.SVG, label string) {
	t := c.Theme()
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		if ic != nil {
			ui.Icon(c, ic).FontSize(14).TextColor(t.TextMuted)
		}
		ui.Text(c, label).FontSize(13).Bold()
	})
}

func (a *App) dashboardPage(c *ui.Context) {
	ui.Column(c).Fill().Padding(16).Gap(10).Children(func() {
		a.dashHeader(c)

		ui.Row(c).Gap(12).AlignItems(ui.Stretch).Children(func() {
			a.statusCard(c)
			a.speedCard(c, "上传", iconUpload, a.up, a.peak(false), a.totalUp, ui.Hex("#7c3aed"), false)
			a.speedCard(c, "下载", iconDownload, a.down, a.peak(true), a.totalDown, ui.Hex("#2563eb"), true)
		})

		ui.Row(c).Gap(12).AlignItems(ui.Stretch).Children(func() {
			a.modeCard(c)
			a.systemProxyCard(c)
			a.tunCard(c)
		})

		ui.Row(c).Gap(12).AlignItems(ui.Stretch).Children(func() {
			a.lanCard(c)
			a.configCard(c)
			a.serviceCard(c)
			a.memoryCard(c)
		})

		ui.Row(c).Gap(12).AlignItems(ui.Stretch).Grow(1).MinHeight(120).Children(func() {
			a.requestsCard(c)
			a.groupsCard(c)
		})
	})
	a.siteModal(c)
}

func (a *App) dashHeader(c *ui.Context) {
	t := c.Theme()
	running := a.running.Load()
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "仪表盘").FontSize(28).Bold()
		ui.Spacer(c)
		button := ui.Button(c, "").Size(56, 56).Radius(28).Padding(0).Label("内核状态")
		if running {
			button.Background(ui.Hex("#22c55e")).TextColor(ui.Hex("#ffffff")).Tooltip("内核运行中，点击停止")
		} else {
			button.Background(t.Surface).TextColor(t.TextMuted).Tooltip("内核已停止，点击启动")
		}
		button.Children(func() {
			if running {
				ui.Icon(c, iconCheck).FontSize(24)
			} else {
				ui.Icon(c, iconPlay).FontSize(20)
			}
		})
		if button.Clicked() {
			if running {
				a.stopCore()
			} else {
				a.startCore()
			}
		}
	})
}

func (a *App) statusCard(c *ui.Context) {
	t := c.Theme()
	running := a.running.Load()
	up := time.Duration(0)
	if !a.startedAt.IsZero() {
		up = time.Since(a.startedAt)
	}
	box := card(c).Grow(2.2)
	box.Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconBolt).FontSize(13).TextColor(t.TextMuted)
			ui.Text(c, "状态").FontSize(12).TextColor(t.TextMuted)
			if running {
				ui.Badge(c, "运行中").Background(ui.Hex("#22c55e").Alpha(0.16)).TextColor(ui.Hex("#16a34a"))
			} else {
				ui.Badge(c, "已停止").Background(t.TextMuted.Alpha(0.14)).TextColor(t.TextMuted)
			}
		})

		ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
			ui.Column(c).Gap(6).Grow(1).Children(func() {
				ui.Text(c, a.lanIP).FontSize(26).Bold().SingleLine().FontFeatures("tnum")
				ui.Row(c).Gap(6).Children(func() {
					ui.Badge(c, "局域网").Background(pillColor(c)).TextColor(t.TextMuted)
					if running {
						ui.Badge(c, "内核在线").Background(t.Accent.Alpha(0.12)).TextColor(t.Accent)
					}
					if node, _, connected := a.nodeStatus(); node != "" {
						switch {
						case connected:
							ui.Badge(c, "节点已连接").Background(ui.Hex("#22c55e").Alpha(0.16)).TextColor(ui.Hex("#16a34a"))
						case a.nodeTested:
							ui.Badge(c, "节点未连接").Background(t.Danger.Alpha(0.14)).TextColor(t.Danger)
						default:
							ui.Badge(c, "节点未检测").Background(t.TextMuted.Alpha(0.14)).TextColor(t.TextMuted)
						}
					}
				})
			})
			// A soft blue blob behind the button, as on FlClash's status card.
			ui.Box(c).Size(110, 110).Center().Draw(func(p *ui.Painter, r ui.Rect) {
				if !running {
					return
				}
				blob := ui.RadialGradient{
					Center: ui.Point{X: 0.5, Y: 0.5},
					Stops: []ui.GradientStop{
						{t.Accent.Alpha(0.20), 0},
						{t.Accent.Alpha(0.05), 0.75},
						{t.Accent.Alpha(0), 1},
					},
				}
				p.FillRadialGradient(ui.Rect{X: r.X + 4, Y: r.Y + 4, W: r.W - 8, H: r.H - 8}, blob, 40)
			}).Children(func() {
				button := ui.Button(c, "").Size(72, 72).Radius(36).Padding(0).Label("启动或停止内核")
				button.Background(t.Accent).TextColor(t.AccentText)
				button.Children(func() {
					if running {
						ui.Icon(c, iconPause).FontSize(24)
					} else {
						ui.Icon(c, iconPlay).FontSize(24)
					}
				})
				if button.Clicked() {
					if running {
						a.stopCore()
					} else {
						a.startCore()
					}
				}
			})
		})

		ui.Row(c).Gap(24).Children(func() {
			info := func(label, value string) {
				ui.Column(c).Gap(2).Children(func() {
					ui.Text(c, label).FontSize(11).TextColor(t.TextMuted)
					ui.Text(c, value).FontSize(13).FontFeatures("tnum").SingleLine()
				})
			}
			uptime := "—"
			if running {
				uptime = formatDuration(up)
			}
			info("运行时长", uptime)
			profile := "未选择"
			if p, ok := a.activeProfile(); ok {
				profile = p.Name
			}
			info("当前配置", profile)
			if node, delay, connected := a.nodeStatus(); node != "" {
				value := node
				if connected {
					value += " · " + delayText(delay)
				}
				info("节点", value)
			}
		})
	})
}

func (a *App) speedCard(c *ui.Context, title string, ic *ui.SVG, current, peak, total int64, col ui.Color, down bool) {
	t := c.Theme()
	box := card(c).Grow(1)
	box.Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, ic).FontSize(13).TextColor(col)
			ui.Text(c, title).FontSize(12).TextColor(t.TextMuted)
			ui.Spacer(c)
			ui.Text(c, "峰值 "+formatSpeed(peak)).FontSize(10).TextColor(t.TextMuted).SingleLine()
		})
		num, unit := splitSpeed(current)
		ui.Row(c).AlignItems(ui.Baseline).Gap(4).Children(func() {
			ui.Text(c, num).FontSize(30).Bold().FontFeatures("tnum").SingleLine()
			ui.Text(c, unit).FontSize(12).TextColor(t.TextMuted)
		})
		ui.Text(c, "累计 "+formatBytes(total)).FontSize(11).TextColor(t.TextMuted).SingleLine()
		ui.Spacer(c)
		sparkline(c, a.history, down, col)
	})
}

func (a *App) modeCard(c *ui.Context) {
	t := c.Theme()
	box := card(c).Grow(2.2)
	box.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconProxies).FontSize(14).TextColor(t.TextMuted)
			ui.Text(c, "出站模式").FontSize(13).Bold()
			ui.Spacer(c)
			a.modeSegmented(c)
		})
	})
}

// modeSegmented builds the outbound mode switch, the chosen segment in the
// accent color as FlClash's dashboard shows it.
func (a *App) modeSegmented(c *ui.Context) {
	t := c.Theme()
	mode := a.mode
	seg := ui.SegmentedBase(c, &mode, 3)
	seg.Track.Padding(3).Radius(10).Background(pillColor(c)).Children(func() {
		for i, label := range []string{"规则", "全局", "直连"} {
			item := seg.Segment(i).Padding(5, 14).Radius(8).Label(label)
			if i == mode {
				item.Background(t.Accent).TextColor(t.AccentText)
			} else {
				item.TextColor(t.TextMuted)
			}
			item.Children(func() {
				ui.Text(c, label).FontSize(12)
			})
		}
	})
	if mode != a.mode {
		a.setMode(modeName(mode))
	}
}

func (a *App) systemProxyCard(c *ui.Context) {
	t := c.Theme()
	on := a.settings.SystemProxy
	state := "已关闭"
	if on {
		state = "已开启"
	}
	box := card(c).Grow(1)
	box.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconLink).FontSize(14).TextColor(t.TextMuted)
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				ui.Text(c, "系统代理").FontSize(13).Bold().SingleLine()
				ui.Text(c, state).FontSize(11).TextColor(t.TextMuted).SingleLine()
			})
			sw := ui.Switch(c, &a.settings.SystemProxy).Label("系统代理")
			if sw.Changed() {
				a.saveSettings(false)
				if a.running.Load() {
					setSystemProxy(a.settings.SystemProxy, a.settings.MixedPort)
				}
			}
		})
	})
}

func (a *App) tunCard(c *ui.Context) {
	t := c.Theme()
	on := a.settings.TunEnabled
	state := "已关闭"
	if on {
		state = "已开启"
	}
	box := card(c).Grow(1)
	box.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconShield).FontSize(14).TextColor(t.TextMuted)
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				ui.Text(c, "虚拟网卡").FontSize(13).Bold().SingleLine()
				ui.Text(c, state).FontSize(11).TextColor(t.TextMuted).SingleLine()
			})
			sw := ui.Switch(c, &a.settings.TunEnabled).Label("虚拟网卡")
			if sw.Changed() {
				a.saveSettings(true)
			}
		})
	})
}

func (a *App) lanCard(c *ui.Context) {
	t := c.Theme()
	box := card(c).Grow(1)
	box.Children(func() {
		cardHead(c, iconServer, "内网 IP")
		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			ui.Text(c, a.lanIP).FontSize(20).Bold().FontFeatures("tnum").Grow(1).SingleLine()
			copy := ui.ButtonBase(c.Key("copy-lan-ip")).Size(28, 28).Radius(6).Center()
			copy.Label("复制内网 IP").Tooltip("复制内网 IP")
			copy.Children(func() {
				ui.Icon(c, iconCopy).FontSize(14).TextColor(t.TextMuted)
			})
			if copy.Clicked() {
				c.WriteClipboard(a.lanIP)
				a.toastText = "已复制 " + a.lanIP
			}
		})
		ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(8).Radius(10).Background(pillColor(c)).Children(func() {
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				ui.Text(c, "局域网代理").FontSize(12).Bold()
				ui.Text(c, map[bool]string{true: "已开启", false: "已关闭"}[a.settings.AllowLAN]).
					FontSize(11).TextColor(t.TextMuted)
			})
			sw := ui.Switch(c, &a.settings.AllowLAN).Label("局域网代理")
			if sw.Changed() {
				a.saveSettings(true)
			}
		})
	})
}

func (a *App) configCard(c *ui.Context) {
	t := c.Theme()
	box := ui.ButtonBase(c.Key("dash-config")).Grow(1).Basis(0).MinWidth(0).Padding(16).Gap(8).Radius(16).
		Background(cardColor(c)).Shadow(0, 1, 3, 0, cardShadow(c)).AlignItems(ui.Stretch)
	box.Children(func() {
		ui.Column(c).Gap(6).Grow(1).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconProfiles).FontSize(14).TextColor(t.TextMuted)
				ui.Text(c, "配置").FontSize(13).Bold()
				ui.Spacer(c)
				ui.Icon(c, iconChevron).FontSize(14).TextColor(t.TextMuted)
			})
			name := "未选择"
			sub := "点击添加配置"
			if p, ok := a.activeProfile(); ok {
				name = p.Name
				sub = "更新于 " + relativeTime(p.UpdatedAt)
			}
			ui.Text(c, name).FontSize(15).Bold().SingleLine()
			ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
			proxies := 0
			for _, g := range a.groups {
				if p := a.proxies[g]; p != nil {
					proxies += len(p.All)
				}
			}
			ui.Row(c).Gap(16).Children(func() {
				count := func(label string, n int) {
					ui.Column(c).Gap(1).Children(func() {
						ui.Text(c, fmt.Sprint(n)).FontSize(14).Bold().FontFeatures("tnum")
						ui.Text(c, label).FontSize(10).TextColor(t.TextMuted)
					})
				}
				count("策略组", len(a.groups))
				count("代理", proxies)
				count("规则", len(a.rules))
			})
		})
	})
	if box.Hovered() {
		box.Background(cardHoverColor(c))
	}
	if box.Clicked() {
		a.route.Push("/profiles")
	}
}

func (a *App) serviceCard(c *ui.Context) {
	t := c.Theme()
	box := ui.ButtonBase(c.Key("dash-service")).Grow(1).Basis(0).MinWidth(0).Padding(16).Gap(8).Radius(16).
		Background(cardColor(c)).Shadow(0, 1, 3, 0, cardShadow(c)).AlignItems(ui.Stretch)
	box.Children(func() {
		ui.Column(c).Gap(8).Grow(1).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconServer).FontSize(14).TextColor(t.TextMuted)
				ui.Text(c, "服务状态").FontSize(13).Bold()
				ui.Spacer(c)
				if a.testingSite {
					ui.Spinner(c).Size(13, 13).Label("检测中")
				} else {
					refresh := ui.ButtonBase(c.Key("service-test")).Size(24, 24).Radius(6).Center()
					refresh.Label("重新检测").Tooltip("重新检测网站连通性")
					refresh.Children(func() {
						ui.Icon(c, iconRefresh).FontSize(13).TextColor(t.TextMuted)
					})
					if refresh.Clicked() {
						a.testSite()
					}
				}
				edit := ui.ButtonBase(c.Key("service-site")).Size(24, 24).Radius(6).Center()
				edit.Label("编辑测试网站").Tooltip("编辑测试网站")
				edit.Children(func() {
					ui.Icon(c, iconEdit).FontSize(13).TextColor(t.TextMuted)
				})
				if edit.Clicked() {
					a.siteDraft = a.settings.TestURL
					a.siteOpen = true
				}
			})
			ui.Text(c, a.settings.TestURL).FontSize(11).TextColor(t.TextMuted).SingleLine()
			ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(6, 8).Radius(8).Background(pillColor(c)).Children(func() {
				dot := ui.Box(c).Size(8, 8).Radius(4).Background(t.TextMuted)
				label := "未检测"
				switch {
				case a.testingSite:
					label = "检测中…"
				case a.nodeTested && !a.nodeOK:
					dot.Background(t.Danger)
					label = "节点未连接"
				case a.running.Load() && a.testOK:
					dot.Background(ui.Hex("#22c55e"))
					label = "网站可达"
				case a.running.Load() && a.nodeOK:
					dot.Background(ui.Hex("#ef4444"))
					label = "网站不可达"
				}
				ui.Text(c, label).FontSize(11).TextColor(t.Text).Grow(1)
				if a.testOK && !a.testingSite {
					ui.Text(c, fmt.Sprintf("%d ms", a.testDelay)).FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
				}
			})
			progress := 0.0
			if a.testOK && a.testDelay > 0 {
				progress = min(1, float64(a.testDelay)/500)
			}
			ui.Progress(c, progress).Label("网站连通性")
		})
	})
	if box.Clicked() {
		a.siteDraft = a.settings.TestURL
		a.siteOpen = true
	}
}

// siteModal edits the test website used by the service card.
func (a *App) siteModal(c *ui.Context) {
	if !a.siteOpen {
		return
	}
	t := c.Theme()
	ui.Modal(c, &a.siteOpen, func() {
		ui.Column(c).Width(440).Padding(20).Gap(12).Radius(14).
			Background(t.Background).Border(1, t.Border).Children(func() {
			ui.Text(c, "编辑测试网站").FontSize(16).Bold()
			ui.Text(c, "服务状态卡片会通过当前选中的代理检测这个地址的连通性。").
				FontSize(11).TextColor(t.TextMuted)
			ui.TextInput(c, &a.siteDraft).Placeholder("http://www.gstatic.com/generate_204").Label("测试网址")
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, "取消").Clicked() {
					a.siteOpen = false
				}
				if ui.PrimaryButton(c, "保存并检测").Clicked() {
					url := strings.TrimSpace(a.siteDraft)
					if url == "" {
						a.toastText = "测试网址不能为空"
						return
					}
					a.settings.TestURL = url
					a.saveSettings(false)
					a.siteOpen = false
					a.testOK, a.testDelay = false, 0
					a.testSite()
				}
			})
		})
	})
}

func (a *App) memoryCard(c *ui.Context) {
	t := c.Theme()
	app := a.appMem
	core := a.memory
	total := app + core
	box := ui.ButtonBase(c.Key("dash-memory")).Grow(1).Basis(0).MinWidth(0).Padding(16).Gap(8).Radius(16).
		Background(cardColor(c)).Shadow(0, 1, 3, 0, cardShadow(c)).AlignItems(ui.Stretch)
	box.Children(func() {
		ui.Column(c).Gap(6).Grow(1).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconChip).FontSize(14).TextColor(t.TextMuted)
				ui.Text(c, "内存信息").FontSize(13).Bold()
				ui.Spacer(c)
				ui.Icon(c, iconChevron).FontSize(14).TextColor(t.TextMuted)
			})
			ui.Text(c, formatBytes(int64(total))).FontSize(20).Bold().FontFeatures("tnum")
			share := 0.0
			if total > 0 {
				share = float64(app) / float64(total)
			}
			ui.Box(c).Height(8).Radius(4).Background(pillColor(c)).Children(func() {
				ui.Box(c).Height(8).Radius(4).WidthPercent(float32(share*100)).
					Gradient(t.Accent, t.AccentHover, 90)
			})
			legend := func(label string, value uint64, col ui.Color) {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(7, 7).Radius(4).Background(col)
					ui.Text(c, label).FontSize(10).TextColor(t.TextMuted).Grow(1)
					ui.Text(c, formatBytes(int64(value))).FontSize(10).TextColor(t.TextMuted).FontFeatures("tnum")
				})
			}
			legend("应用", app, t.Accent)
			legend("内核", core, ui.Hex("#a5b4fc"))
		})
	})
	if box.Hovered() {
		box.Background(cardHoverColor(c))
	}
	if box.Clicked() {
		a.route.Push("/tools/settings")
	}
}

func (a *App) requestsCard(c *ui.Context) {
	t := c.Theme()
	// Match the status card's share of the row: 2.2 of 4.2.
	box := card(c).Grow(2.2)
	box.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			a.dashTabs(c)
			ui.Spacer(c)
			ui.Text(c, fmt.Sprint(len(a.conns))+" 条").FontSize(11).TextColor(t.TextMuted)
		})
		switch a.dashTab {
		case 0:
			cols := []ui.TableColumn{
				{Title: "来源", Width: 120},
				{Title: "目标", ID: "target"},
				{Title: "出口", Width: 110},
				{Title: "下载", Width: 65, Align: ui.End},
				{Title: "上传", Width: 65, Align: ui.End},
			}
			ui.Table(c, &a.dashConn, cols, len(a.conns), func(row, col int) {
				conn := a.conns[row]
				switch col {
				case 0:
					src := conn.Metadata.SourceIP
					if conn.Metadata.SourcePort != "" {
						src += ":" + conn.Metadata.SourcePort
					}
					ui.Text(c, src).FontSize(11).TextColor(t.TextMuted).SingleLine()
				case 1:
					target := conn.Metadata.Host
					if target == "" {
						target = conn.Metadata.Destination
					}
					if conn.Metadata.DstPort != "" {
						target += ":" + conn.Metadata.DstPort
					}
					ui.Text(c, target).FontSize(11).SingleLine()
				case 2:
					exit := ""
					if len(conn.Chains) > 0 {
						exit = conn.Chains[len(conn.Chains)-1]
					}
					ui.Text(c, exit).FontSize(11).TextColor(t.TextMuted).SingleLine()
				case 3:
					ui.Text(c, formatBytes(conn.Download)).FontSize(11).FontFeatures("tnum").SingleLine()
				case 4:
					ui.Text(c, formatBytes(conn.Upload)).FontSize(11).FontFeatures("tnum").SingleLine()
				}
			}).Grow(1).Children(func() {
				if len(a.conns) == 0 {
					ui.Text(c, "暂无数据").FontSize(12).TextColor(t.TextMuted).Padding(16).TextAlign(ui.Center)
				}
			})
		case 1:
			ui.List(c, &a.dashList, len(a.recent), func(i int) {
				r := a.recent[i]
				ui.Row(c).Gap(8).Padding(4, 6).Children(func() {
					ui.Text(c, r.Host).FontSize(11).Grow(1).SingleLine()
					ui.Text(c, strings.Join(r.Chains, " → ")).FontSize(10).TextColor(t.TextMuted).Width(110).SingleLine()
					ui.Text(c, formatBytes(r.Down)).FontSize(10).TextColor(t.TextMuted).FontFeatures("tnum").Width(60).TextAlign(ui.End)
					ui.Text(c, relativeTime(r.At)).FontSize(10).TextColor(t.TextMuted).Width(70).TextAlign(ui.End)
				})
			}).Grow(1).Children(func() {
				if len(a.recent) == 0 {
					ui.Text(c, "暂无数据").FontSize(12).TextColor(t.TextMuted).Padding(16).TextAlign(ui.Center)
				}
			})
		default:
			ui.Column(c).Grow(1).Center().Children(func() {
				ui.Text(c, "暂无 DNS 查询数据").FontSize(12).TextColor(t.TextMuted)
			})
		}
	})
}

func (a *App) groupsCard(c *ui.Context) {
	t := c.Theme()
	box := ui.ButtonBase(c.Key("dash-groups")).Grow(1.35).Basis(0).MinWidth(0).Padding(18).Gap(8).Radius(20).
		Background(cardColor(c)).Border(1, cardEdge(c)).Shadow(0, 2, 8, 0, cardShadow(c)).AlignItems(ui.Stretch).
		Transition(ui.ElementTransition{Colors: true})
	box.Children(func() {
		ui.Column(c).Gap(8).Grow(1).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconRules).FontSize(14).TextColor(t.TextMuted)
				ui.Text(c, "策略组").FontSize(13).Bold()
				ui.Badge(c, fmt.Sprint(len(a.groups))).Background(pillColor(c)).TextColor(t.TextMuted)
				ui.Spacer(c)
				ui.Icon(c, iconChevron).FontSize(14).TextColor(t.TextMuted)
			})
			ui.Scroll(c).Grow(1).ScrollbarAxes(true, false).Children(func() {
				ui.Column(c).Gap(6).Children(func() {
					for _, name := range a.groups {
						g := a.proxies[name]
						now := "未选择"
						if g != nil && g.Now != "" {
							now = g.Now
						}
						ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(8, 10).Radius(10).
							Background(pillColor(c)).Children(func() {
							ui.Box(c).Size(26, 26).Radius(13).Background(t.Surface).Center().Children(func() {
								ui.Text(c, firstLetter(name)).FontSize(12).Bold().TextColor(t.TextMuted)
							})
							ui.Column(c).Gap(1).Grow(1).Children(func() {
								ui.Text(c, name).FontSize(12).Bold().SingleLine()
								ui.Text(c, now).FontSize(10).TextColor(t.TextMuted).SingleLine()
							})
						})
					}
					if len(a.groups) == 0 {
						ui.Text(c, "暂无策略组").FontSize(12).TextColor(t.TextMuted).Padding(12)
					}
				})
			})
		})
	})
	if box.Hovered() {
		box.Background(cardHoverColor(c))
	}
	if box.Clicked() {
		a.route.Push("/proxies")
	}
}

// peak returns the largest rate of the last minute, download or upload.
func (a *App) peak(down bool) int64 {
	var peak int64
	for _, s := range a.history {
		v := s.up
		if down {
			v = s.down
		}
		if v > peak {
			peak = v
		}
	}
	return peak
}

// splitSpeed splits a rate into its number and its unit, as the dashboard
// shows them at two sizes.
func splitSpeed(n int64) (string, string) {
	s := formatBytes(n)
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return s, ""
	}
	return parts[0], parts[1] + "/s"
}

// relativeTime writes how long ago something happened.
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(d.Hours()))
	default:
		return fmt.Sprintf("%d 天前", int(d.Hours()/24))
	}
}

// firstLetter returns the first letter of s, upper case, for a badge.
func firstLetter(s string) string {
	for _, r := range s {
		return strings.ToUpper(string(r))
	}
	return "?"
}

// sparkline draws the last minute of traffic as a small curve, as FlClash
// puts at the foot of its speed cards.
func sparkline(c *ui.Context, values []sample, down bool, col ui.Color) {
	ui.Box(c).Height(36).Draw(func(p *ui.Painter, r ui.Rect) {
		r.X += r.W * 0.3
		r.W *= 0.7
		if len(values) < 2 {
			return
		}
		var max float32
		for _, s := range values {
			v := s.up
			if down {
				v = s.down
			}
			if float32(v) > max {
				max = float32(v)
			}
		}
		if max <= 0 {
			return
		}
		pts := make([][2]float32, len(values))
		for i, s := range values {
			v := s.up
			if down {
				v = s.down
			}
			pts[i] = [2]float32{
				r.X + float32(i)*r.W/float32(len(values)-1),
				r.Y + r.H - (float32(v)/max)*r.H*0.85 - r.H*0.08,
			}
		}
		area := &ui.Path{}
		area.MoveTo(pts[0][0], r.Y+r.H)
		for _, pt := range pts {
			area.LineTo(pt[0], pt[1])
		}
		area.LineTo(pts[len(pts)-1][0], r.Y+r.H)
		area.Close()
		p.FillPathGradient(area, ui.LinearGradient{
			Angle: 180,
			Stops: []ui.GradientStop{{col.Alpha(0.26), 0}, {col.Alpha(0.02), 1}},
		})
		p.StrokePolyline(pts, 2, col)
	})
}

// dashTabs builds the tabs of the connections card, the chosen one in the
// accent color with its count, as FlClash's dashboard shows them.
func (a *App) dashTabs(c *ui.Context) {
	t := c.Theme()
	counts := []int{len(a.conns), len(a.recent), 0}
	labels := []string{"实时连接", "最近请求", "DNS 查询"}
	tabs := ui.TabsBase(c, &a.dashTab, len(labels))
	tabs.List.Gap(6).Children(func() {
		for i, label := range labels {
			tab := tabs.Tab(i).Padding(4, 12).Radius(8).Label(label)
			if i == a.dashTab {
				tab.Background(t.Accent).TextColor(t.AccentText)
			} else {
				tab.Background(pillColor(c)).TextColor(t.TextMuted)
			}
			tab.Children(func() {
				ui.Textf(c, "%d %s", counts[i], label).FontSize(12)
			})
		}
	})
}
