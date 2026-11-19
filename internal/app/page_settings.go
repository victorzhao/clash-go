package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/victorzhao/clash-go/internal/platform"
)

func (a *App) settingsPage(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().ScrollbarAxes(true, false).Children(func() {
		ui.Column(c).Padding(24).Gap(14).Children(func() {
			a.pageTitle(c, "设置", "修改后立即保存；需要重载的设置会重启内核")

			a.section(c, "内核", "监听端口、分流模式与运行方式", func() {
				ui.Row(c).Gap(16).Children(func() {
					ui.Column(c).Gap(4).Grow(1).Children(func() {
						ui.Text(c, "代理模式").FontSize(12).Bold()
						a.modeSegmented(c)
					})
					ui.Column(c).Gap(4).Width(140).Children(func() {
						ui.Text(c, "日志级别").FontSize(12).Bold()
						if ui.Select(c, &a.settings.LogLevel, []string{"info", "warning", "debug", "error"}).Changed() {
							a.saveSettings(true)
						}
					})
				})

				ui.Row(c).Gap(16).Children(func() {
					ui.Column(c).Gap(4).Grow(1).Children(func() {
						ui.Text(c, "混合端口").FontSize(12).Bold()
						ui.Text(c, "HTTP 与 SOCKS 共用，系统代理指向它").FontSize(11).TextColor(t.TextMuted)
						if ui.NumberInput(c, &a.portDraft, 1024, 65535, 1).Changed() {
							a.saveSettings(true)
						}
					})
					ui.Column(c).Gap(4).Grow(1).Children(func() {
						ui.Text(c, "API 端口").FontSize(12).Bold()
						ui.Text(c, "只监听 127.0.0.1，界面通过它读数据").FontSize(11).TextColor(t.TextMuted)
						if ui.NumberInput(c, &a.apiPortDraft, 1024, 65535, 1).Changed() {
							a.saveSettings(true)
						}
					})
				})

				if a.switchRow(c, "允许局域网连接", "同一网络的其他设备可以通过本机代理上网", &a.settings.AllowLAN) {
					a.saveSettings(true)
				}
				if a.switchRow(c, "启用 IPv6", "让内核解析并路由 IPv6 流量", &a.settings.IPv6) {
					a.saveSettings(true)
				}
				if a.switchRow(c, "TUN 模式", "接管所有流量，Windows 上需要管理员权限和 wintun.dll", &a.settings.TunEnabled) {
					a.saveSettings(true)
				}
				if a.settings.TunEnabled {
					ui.Column(c).Gap(4).Width(160).Children(func() {
						ui.Text(c, "TUN 栈").FontSize(12).Bold()
						if ui.Select(c, &a.settings.TunStack, []string{"mixed", "gvisor", "system"}).Changed() {
							a.saveSettings(true)
						}
					})
				}
			})

			a.section(c, "系统", "开机启动、系统代理与外观", func() {
				if a.switchRow(c, "设置系统代理", "内核运行时把 Windows 的系统代理指向混合端口，停止时恢复", &a.settings.SystemProxy) {
					a.saveSettings(false)
					if a.running.Load() {
						platform.SetSystemProxy(a.settings.SystemProxy, fmt.Sprintf("127.0.0.1:%d", a.settings.MixedPort))
					}
				}
				if a.switchRow(c, "开机自动启动", "登录 Windows 时启动 Clash Go", &a.settings.AutoStart) {
					a.saveSettings(false)
				}
				if a.switchRow(c, "打开应用时启动内核", "打开窗口后直接按当前配置运行内核", &a.settings.StartCore) {
					a.saveSettings(false)
				}

				ui.Row(c).Gap(16).Children(func() {
					ui.Column(c).Gap(4).Width(160).Children(func() {
						ui.Text(c, "外观").FontSize(12).Bold()
						themeLabel := map[string]string{"system": "跟随系统", "light": "浅色", "dark": "深色"}[a.settings.Theme]
						if themeLabel == "" {
							themeLabel = "跟随系统"
						}
						if ui.Select(c, &themeLabel, []string{"跟随系统", "浅色", "深色"}).Changed() {
							a.settings.Theme = map[string]string{"跟随系统": "system", "浅色": "light", "深色": "dark"}[themeLabel]
							a.saveSettings(false)
						}
					})
					ui.Column(c).Gap(4).Width(160).Children(func() {
						ui.Text(c, "语言").FontSize(12).Bold()
						if ui.Select(c, &a.settings.Language, []string{"zh-CN", "zh-TW", "en"}).Changed() {
							a.saveSettings(false)
						}
					})
				})
			})

			a.section(c, "配置", "当前使用的配置与订阅更新", func() {
				names := make([]string, 0, len(a.profiles)+1)
				names = append(names, "未选择")
				ids := []string{""}
				at := 0
				for i, p := range a.profiles {
					names = append(names, p.Name)
					ids = append(ids, p.ID)
					if p.ID == a.settings.ActiveProfile {
						at = i + 1
					}
				}
				ui.Column(c).Gap(4).Width(260).Children(func() {
					ui.Text(c, "当前配置").FontSize(12).Bold()
					chosen := names[at]
					if ui.Select(c, &chosen, names).Changed() {
						for i, n := range names {
							if n == chosen {
								a.settings.ActiveProfile = ids[i]
								a.saveSettings(true)
								break
							}
						}
					}
				})

				ui.Column(c).Gap(4).Width(260).Children(func() {
					ui.Text(c, "自动更新订阅").FontSize(12).Bold()
					options := []string{"关闭", "每 6 小时", "每 12 小时", "每 24 小时"}
					idx := intervalIndex(a.settings.UpdateInterval)
					chosen := options[idx]
					if ui.Select(c, &chosen, options).Changed() {
						for i, o := range options {
							if o == chosen {
								a.settings.UpdateInterval = intervalHours(i)
								a.saveSettings(false)
								break
							}
						}
					}
				})
			})

			a.section(c, "关于", "", func() {
				ui.Row(c).Gap(20).Children(func() {
					info := func(label, value string) {
						ui.Column(c).Gap(2).Children(func() {
							ui.Text(c, label).FontSize(11).TextColor(t.TextMuted)
							ui.Text(c, value).FontSize(12).Selectable()
						})
					}
					version := a.version
					if version == "" {
						version = "—"
					}
					info("Clash Go", "1.0.0")
					info("mihomo 内核", version)
					info("配置目录", a.st.Dir())
				})
			})
		})
	})
}

// switchRow builds a setting with its switch and description, reporting a
// change.
func (a *App) switchRow(c *ui.Context, title, desc string, on *bool) bool {
	t := c.Theme()
	changed := false
	ui.Column(c).Gap(2).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			sw := ui.Switch(c, on).Label(title)
			if sw.Changed() {
				changed = true
			}
			ui.Text(c, title).FontSize(12).Bold()
		})
		if desc != "" {
			ui.Text(c, desc).FontSize(11).TextColor(t.TextMuted)
		}
	})
	return changed
}

// section builds a card of settings.
func (a *App) section(c *ui.Context, title, desc string, fn func()) {
	t := c.Theme()
	card := ui.Column(c).Padding(18).Gap(10).Radius(16).
		Background(cardColor(c)).Border(1, cardEdge(c)).Shadow(0, 2, 8, 0, cardShadow(c))
	card.Children(func() {
		ui.Text(c, title).FontSize(14).Bold()
		if desc != "" {
			ui.Text(c, desc).FontSize(11).TextColor(t.TextMuted)
		}
		fn()
	})
}

func intervalIndex(hours int) int {
	switch hours {
	case 6:
		return 1
	case 12:
		return 2
	case 24:
		return 3
	}
	return 0
}

func intervalHours(i int) int {
	switch i {
	case 1:
		return 6
	case 2:
		return 12
	case 3:
		return 24
	}
	return 0
}
