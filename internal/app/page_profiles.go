package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/victorzhao/clash-go/internal/profile"
	"github.com/victorzhao/clash-go/internal/store"
)

func (a *App) profilesPage(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().ScrollbarAxes(true, false).Children(func() {
		ui.Column(c).Padding(24).Gap(12).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
				a.pageTitle(c, "配置", "订阅与本地配置文件")
				ui.Spacer(c)
				if ui.PrimaryButton(c, "添加配置").Clicked() {
					a.addOpen = true
					a.addTab = 0
					a.addName, a.addURL, a.addText = "", "", ""
				}
			})

			if len(a.profiles) == 0 {
				ui.Column(c).Padding(24).Gap(6).Radius(12).Center().
					Background(t.Surface).Border(1, t.Border).Children(func() {
					ui.Text(c, "还没有配置").FontSize(14).Bold()
					ui.Text(c, "点击「添加配置」，从订阅链接、本地文件或粘贴的 YAML 导入。").
						FontSize(12).TextColor(t.TextMuted).TextAlign(ui.Center)
				})
			}

			for _, p := range a.profiles {
				a.profileCard(c, p)
			}
		})
	})
	a.addProfileModal(c)
}

func (a *App) profileCard(c *ui.Context, p store.Profile) {
	t := c.Theme()
	active := p.ID == a.settings.ActiveProfile
	card := ui.Column(c).Padding(16).Gap(8).Radius(16).
		Background(cardColor(c)).Border(1, cardEdge(c)).Shadow(0, 2, 8, 0, cardShadow(c))
	if active {
		card.Border(1, t.Accent.Alpha(0.55))
	}
	card.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Icon(c, iconProfiles).FontSize(15).TextColor(t.TextMuted)
			ui.Text(c, p.Name).FontSize(14).Bold().Grow(1).SingleLine()
			if active {
				ui.Badge(c, "使用中").Background(t.Accent.Alpha(0.16)).TextColor(t.Accent)
			}
			ui.Text(c, map[string]string{"remote": "订阅", "local": "本地"}[p.Type]).
				FontSize(11).TextColor(t.TextMuted)
		})
		sub := "更新于 " + formatTime(p.UpdatedAt)
		if p.URL != "" {
			sub += " · " + p.URL
		}
		ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
		if p.Total > 0 {
			ui.Column(c).Gap(4).Children(func() {
				ui.Progress(c, float64(p.Used)/float64(p.Total)).Label("订阅流量")
				line := fmt.Sprintf("已用 %s / %s", formatBytes(p.Used), formatBytes(p.Total))
				if !p.Expire.IsZero() {
					line += " · 到期 " + p.Expire.Local().Format("2006-01-02")
				}
				ui.Text(c, line).FontSize(11).TextColor(t.TextMuted)
			})
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "查看").Clicked() {
				a.route.Push("/profiles/" + p.ID)
			}
			if !active {
				if ui.Button(c, "启用").Clicked() {
					a.activateProfile(p)
				}
			}
			if p.Type == "remote" {
				if ui.Button(c, "更新").Clicked() {
					a.updateProfile(p)
				}
			}
			if ui.Button(c, "删除").Clicked() {
				a.removeProfile(p)
			}
			ui.Spacer(c)
			ui.Text(c, p.ID).FontSize(10).TextColor(t.TextMuted).Selectable()
		})
	})
}

func (a *App) activateProfile(p store.Profile) {
	a.settings.ActiveProfile = p.ID
	if err := a.st.SaveSettings(a.settings); err != nil {
		a.toastText = "保存失败：" + err.Error()
		return
	}
	if a.running.Load() {
		a.restartCore()
		a.toastText = "已切换到「" + p.Name + "」"
		return
	}
	a.toastText = "已选择「" + p.Name + "」，启动内核后生效"
}

func (a *App) addProfileModal(c *ui.Context) {
	if !a.addOpen {
		return
	}
	t := c.Theme()
	ui.Modal(c, &a.addOpen, func() {
		ui.Column(c).Width(540).Padding(20).Gap(12).Radius(14).
			Background(t.Background).Border(1, t.Border).Children(func() {
			ui.Text(c, "添加配置").FontSize(16).Bold()
			ui.Tabs(c, &a.addTab, "订阅链接", "本地文件", "手动输入")
			switch a.addTab {
			case 0:
				ui.TextInput(c, &a.addName).Placeholder("配置名称，留空则用域名").Label("名称")
				ui.TextInput(c, &a.addURL).Placeholder("https://example.com/subscription").Label("订阅链接")
				ui.Text(c, "支持 Clash YAML 订阅，或返回 vless:// 等分享链接（含 base64）的地址。").
					FontSize(11).TextColor(t.TextMuted)
				ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
					if ui.Button(c, "取消").Clicked() {
						a.addOpen = false
					}
					if ui.PrimaryButton(c, "导入").Disabled(a.adding).Clicked() {
						a.importFromURL()
					}
				})
			case 1:
				ui.Text(c, "选择本机的 Clash / mihomo 配置（YAML）。").
					FontSize(12).TextColor(t.TextMuted)
				ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
					if ui.Button(c, "取消").Clicked() {
						a.addOpen = false
					}
					if ui.PrimaryButton(c, "选择文件…").Clicked() {
						a.importFromFile()
					}
				})
			case 2:
				ui.TextInput(c, &a.addName).Placeholder("配置名称").Label("名称")
				ui.TextArea(c, &a.addText).Placeholder("粘贴 Clash / mihomo 的 YAML，或 vless:// 等分享链接").Height(220).Label("内容")
				ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
					if ui.Button(c, "取消").Clicked() {
						a.addOpen = false
					}
					if ui.PrimaryButton(c, "保存").Clicked() {
						a.importFromText()
					}
				})
			}
		})
	})
}

func (a *App) importFromURL() {
	url := strings.TrimSpace(a.addURL)
	if url == "" {
		a.toastText = "请填写订阅链接"
		return
	}
	if a.adding {
		return
	}
	a.adding = true
	go func() {
		data, info, err := profile.Fetch(url)
		a.update(func() {
			a.adding = false
			if err != nil {
				a.toastText = "下载失败：" + err.Error()
				return
			}
			name := strings.TrimSpace(a.addName)
			if name == "" {
				name = urlHost(url)
			}
			a.addProfile(name, "remote", url, data, info)
			a.addOpen = false
			a.addName, a.addURL = "", ""
			if !a.running.Load() {
				a.startCore()
			}
		})
	}()
}

func (a *App) importFromFile() {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:  a.win,
			Title:   "选择配置文件",
			Filters: []mygo.FileFilter{{Name: "Clash 配置", Extensions: []string{"yaml", "yml", "txt"}}},
		})
		if err != nil || len(paths) == 0 {
			return
		}
		data, err := os.ReadFile(paths[0])
		a.update(func() {
			if err != nil {
				a.toastText = "读取文件失败：" + err.Error()
				return
			}
			prepared, err := profile.Prepare(data)
			if err != nil {
				a.toastText = "不是有效的配置：" + err.Error()
				return
			}
			name := strings.TrimSuffix(filepath.Base(paths[0]), filepath.Ext(paths[0]))
			a.addProfile(name, "local", "", prepared, profile.Info{})
			a.addOpen = false
		})
	}()
}

func (a *App) importFromText() {
	data := []byte(a.addText)
	if len(data) == 0 {
		a.toastText = "请粘贴配置内容"
		return
	}
	prepared, err := profile.Prepare(data)
	if err != nil {
		a.toastText = "不是有效的配置：" + err.Error()
		return
	}
	name := strings.TrimSpace(a.addName)
	if name == "" {
		if lines := strings.Split(strings.TrimSpace(a.addText), "\n"); len(lines) > 0 {
			if i := strings.Index(lines[0], "#"); i >= 0 && i+1 < len(lines[0]) {
				name = strings.TrimSpace(lines[0][i+1:])
			}
		}
	}
	if name == "" {
		name = fmt.Sprintf("本地配置 %s", time.Now().Format("01-02 15:04"))
	}
	a.addProfile(name, "local", "", prepared, profile.Info{})
	a.addOpen = false
	a.addName, a.addText = "", ""
}

func urlHost(u string) string {
	u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	if i := strings.IndexAny(u, "/?#"); i >= 0 {
		u = u[:i]
	}
	return u
}

// profileViewPage shows a profile's configuration, read-only, with the
// actions the profile's card has.
func (a *App) profileViewPage(c *ui.Context, id string) {
	t := c.Theme()
	p, ok := a.findProfile(id)
	if !ok {
		ui.Column(c).Fill().Center().Gap(8).Children(func() {
			ui.Text(c, "找不到这个配置").FontSize(16).Bold()
			if ui.Button(c, "返回配置").Clicked() {
				a.route.Replace("/profiles")
			}
		})
		return
	}
	if a.viewID != id {
		a.viewID = id
		a.viewEditor = nil
		a.viewErr = ""
		a.viewText = ""
		data, err := a.st.ProfileContent(id)
		if err != nil {
			a.viewErr = err.Error()
		} else {
			buf := editor.NewBuffer(string(data))
			buf.SetLanguage(editor.LanguageFor(p.Name + ".yaml"))
			a.viewEditor = &editor.State{Buffer: buf, ReadOnly: true, SoftWrap: true, Label: "配置内容"}
			a.viewText = string(data)
		}
	}

	lines := 0
	if a.viewText != "" {
		lines = strings.Count(a.viewText, "\n") + 1
	}
	meta := map[string]string{"remote": "订阅", "local": "本地"}[p.Type]
	meta += " · " + fmt.Sprintf("%d 行 · %s · 更新于 %s", lines, formatBytes(int64(len(a.viewText))), formatTime(p.UpdatedAt))
	if p.URL != "" {
		meta += " · " + p.URL
	}

	ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.BackButton(c, a.route).Label("返回")
			ui.Text(c, p.Name).FontSize(20).Bold().Grow(1).MinWidth(0).SingleLine()
			if p.ID != a.settings.ActiveProfile {
				if ui.Button(c, "启用").Clicked() {
					a.activateProfile(p)
				}
			}
			if p.Type == "remote" {
				if ui.Button(c, "更新").Clicked() {
					a.updateProfile(p)
				}
			}
			if ui.Button(c, "复制").Clicked() {
				c.WriteClipboard(a.viewText)
				a.toastText = "已复制配置内容"
			}
			if ui.Button(c, "删除").Clicked() {
				a.removeProfile(p)
				a.route.Replace("/profiles")
			}
		})
		ui.Row(c).Clip().Children(func() {
			ui.Text(c, meta).FontSize(11).TextColor(t.TextMuted).SingleLine()
		})

		if a.viewEditor == nil {
			ui.Column(c).Grow(1).Center().Gap(8).Radius(10).
				Background(cardColor(c)).Border(1, t.Border).Children(func() {
				ui.Text(c, "读取配置失败").Bold().TextColor(t.Danger)
				ui.Text(c, a.viewErr).FontSize(12).TextColor(t.TextMuted).Selectable()
				if ui.Button(c, "重试").Clicked() {
					a.viewID = ""
				}
			})
			return
		}
		editor.View(c, a.viewEditor).Grow(1).FillWidth().Radius(10).
			Background(cardColor(c)).Border(1, t.Border)
	})
}
