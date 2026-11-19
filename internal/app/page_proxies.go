package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *App) proxiesPage(c *ui.Context) {
	t := c.Theme()
	if !a.running.Load() {
		a.emptyState(c, "代理", "内核未运行，启动后才能查看和切换代理。", true)
		return
	}
	ui.Row(c).Fill().Children(func() {
		ui.Column(c).Width(240).FillHeight().Padding(16).Gap(8).
			Background(t.Surface).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, "代理组").FontSize(11).Bold().TextColor(t.TextMuted)
			if len(a.groups) == 0 {
				ui.Text(c, "配置中没有代理组").FontSize(12).TextColor(t.TextMuted).Padding(0, 6)
				return
			}
			ui.Scroll(c).Grow(1).ScrollbarAxes(true, false).Children(func() {
				ui.Column(c).Gap(4).Children(func() {
					for i, name := range a.groups {
						g := a.proxies[name]
						item := ui.ButtonBase(c.Key("group"+name)).Padding(8, 10).Radius(8).Gap(8)
						switch {
						case i == a.groupAt:
							item.Background(t.Accent.Alpha(0.14)).TextColor(t.Accent)
						case item.Hovered():
							item.Background(pillColor(c)).TextColor(t.Text)
						default:
							item.TextColor(t.Text)
						}
						item.Children(func() {
							ui.Column(c).Gap(2).Grow(1).Children(func() {
								ui.Text(c, name).FontSize(13).SingleLine()
								now := "未选择"
								if g != nil && g.Now != "" {
									now = g.Now
								}
								ui.Text(c, now).FontSize(11).TextColor(t.TextMuted).SingleLine()
							})
							if g != nil {
								ui.Text(c, fmt.Sprint(len(g.All))).FontSize(11).TextColor(t.TextMuted)
							}
						})
						if item.Clicked() {
							a.groupAt = i
						}
					}
				})
			})
		})

		ui.Column(c).Grow(1).FillHeight().Padding(20).Gap(12).Children(func() {
			g := a.group()
			if g == nil {
				ui.Text(c, "选择一个代理组").TextColor(t.TextMuted)
				return
			}
			ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
				a.pageTitle(c, g.Name, fmt.Sprintf("%s · %d 个代理", g.Type, len(g.All)))
				ui.Spacer(c)
				if a.testing {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Spinner(c).Size(14, 14)
						ui.Text(c, "测速中…").FontSize(12).TextColor(t.TextMuted)
					})
				} else if ui.Button(c, "测速").Clicked() {
					a.testGroup()
				}
			})

			ui.Scroll(c).Grow(1).ScrollbarAxes(true, false).Children(func() {
				ui.Column(c).Gap(6).Children(func() {
					for _, name := range g.All {
						p := a.proxies[name]
						kind := ""
						delay := 0
						if p != nil {
							kind = p.Type
							delay = p.Delay
						}
						row := ui.ButtonBase(c.Key("proxy"+name)).Padding(10, 12).Radius(9).
							Background(t.Surface).Border(1, t.Border).Gap(12)
						row.Children(func() {
							ui.Column(c).Gap(2).Grow(1).Children(func() {
								ui.Text(c, name).FontSize(13).SingleLine()
								ui.Text(c, kind).FontSize(11).TextColor(t.TextMuted)
							})
							if name == g.Now {
								ui.Badge(c, "当前").Background(t.Accent.Alpha(0.16)).TextColor(t.Accent)
							}
							ui.Text(c, delayText(delay)).FontSize(11).TextColor(delayColor(t, delay)).FontFeatures("tnum")
						})
						if row.Hovered() {
							row.Background(cardHoverColor(c))
						}
						if row.Clicked() && name != g.Now {
							a.selectProxy(name)
						}
					}
				})
			})
		})
	})
}

// emptyState shows a page that has nothing to show yet.
func (a *App) emptyState(c *ui.Context, title, message string, start bool) {
	t := c.Theme()
	ui.Column(c).Fill().Center().Gap(10).Padding(24).Children(func() {
		ui.Text(c, title).FontSize(22).Bold()
		ui.Text(c, message).FontSize(13).TextColor(t.TextMuted).TextAlign(ui.Center).MaxWidth(420)
		if start && ui.PrimaryButton(c, "启动内核").Clicked() {
			a.startCore()
		}
	})
}
