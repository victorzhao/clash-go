package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/victorzhao/clash-go/internal/mihomoapi"
)

func (a *App) rulesPage(c *ui.Context) {
	t := c.Theme()
	if !a.running.Load() {
		// The core may have failed to start because of a bad rule. Read the
		// rules from the profile on disk so they can still be edited.
		a.loadRules()
	}
	subtitle := fmt.Sprintf("当前配置的 %d 条规则，按顺序匹配", len(a.rules))
	if !a.running.Load() {
		subtitle = "内核未运行 · 显示配置文件中的规则，修改后可从仪表盘启动内核"
	}
	ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
			a.pageTitle(c, "规则", subtitle)
			ui.Spacer(c)
			if ui.PrimaryButton(c, "添加规则").Clicked() {
				a.ruleEditAt = -1
				a.ruleType, a.rulePayload, a.ruleProxy = "DOMAIN-SUFFIX", "", ""
				a.ruleOpen = true
			}
		})

		a.defaultPolicyCard(c)

		cols := []ui.TableColumn{
			{Title: "类型", Width: 150},
			{Title: "内容", ID: "payload"},
			{Title: "策略", Width: 200},
			{Title: "操作", Width: 90},
		}
		ui.Column(c).Grow(1).Radius(16).Background(cardColor(c)).
			Border(1, cardEdge(c)).Shadow(0, 2, 8, 0, cardShadow(c)).Padding(6).Children(func() {
			ui.Table(c, &a.ruleList, cols, len(a.rules), func(row, col int) {
				rule := a.rules[row]
				switch col {
				case 0:
					ui.Text(c, rule.Type).TextColor(t.Accent).SingleLine()
				case 1:
					ui.Text(c, rule.Payload).SingleLine().Tooltip(rule.Payload)
				case 2:
					ui.Text(c, rule.Proxy).SingleLine()
				case 3:
					ui.Row(c).Gap(4).Children(func() {
						edit := ui.ButtonBase(c.Key(fmt.Sprintf("rule-edit-%d", row))).Size(26, 26).Radius(6).Center()
						edit.Label("编辑规则").Tooltip("编辑")
						edit.Children(func() { ui.Icon(c, iconEdit).FontSize(13).TextColor(t.TextMuted) })
						if edit.Clicked() {
							a.ruleEditAt = row
							a.ruleType, a.rulePayload, a.ruleProxy = ruleTypeConfig(rule.Type), rule.Payload, rule.Proxy
							a.ruleOpen = true
						}
						del := ui.ButtonBase(c.Key(fmt.Sprintf("rule-del-%d", row))).Size(26, 26).Radius(6).Center()
						del.Label("删除规则").Tooltip("删除")
						del.Children(func() { ui.Icon(c, iconTrash).FontSize(13).TextColor(t.TextMuted) })
						if del.Clicked() {
							next := make([]mihomoapi.Rule, 0, len(a.rules)-1)
							next = append(next, a.rules[:row]...)
							next = append(next, a.rules[row+1:]...)
							a.saveRules(next)
						}
					})
				}
			}).Grow(1)
		})
	})
	a.ruleModal(c)
}

func (a *App) ruleModal(c *ui.Context) {
	if !a.ruleOpen {
		return
	}
	t := c.Theme()
	ui.Modal(c, &a.ruleOpen, func() {
		ui.Column(c).Width(500).Padding(20).Gap(12).Radius(14).
			Background(t.Background).Border(1, t.Border).Children(func() {
			title := "添加规则"
			if a.ruleEditAt >= 0 {
				title = "编辑规则"
			}
			ui.Text(c, title).FontSize(16).Bold()
			ui.Row(c).Gap(12).Children(func() {
				ui.Column(c).Gap(4).Width(180).Children(func() {
					ui.Text(c, "类型").FontSize(12).Bold()
					ui.Select(c, &a.ruleType, ruleTypes)
				})
				ui.Column(c).Gap(4).Grow(1).Children(func() {
					ui.Text(c, "策略").FontSize(12).Bold()
					ui.Select(c, &a.ruleProxy, a.ruleProxies())
				})
			})
			ui.Column(c).Gap(4).Children(func() {
				ui.Text(c, "内容").FontSize(12).Bold()
				ui.TextInput(c, &a.rulePayload).Placeholder("example.com")
			})
			ui.Text(c, "规则按顺序匹配；类型与内容用逗号组成一条规则。").
				FontSize(11).TextColor(t.TextMuted)
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, "取消").Clicked() {
					a.ruleOpen = false
				}
				if ui.PrimaryButton(c, "保存").Clicked() {
					kind := ruleTypeConfig(strings.TrimSpace(a.ruleType))
					payload := strings.TrimSpace(a.rulePayload)
					proxy := strings.TrimSpace(a.ruleProxy)
					if kind == "" || (payload == "" && kind != "MATCH") {
						a.toastText = "类型和内容不能为空"
						return
					}
					if proxy == "" {
						a.toastText = "请选择策略"
						return
					}
					r := mihomoapi.Rule{Type: kind, Payload: payload, Proxy: proxy}
					next := make([]mihomoapi.Rule, 0, len(a.rules)+1)
					if a.ruleEditAt >= 0 && a.ruleEditAt < len(a.rules) {
						next = append(next, a.rules[:a.ruleEditAt]...)
						next = append(next, r)
						next = append(next, a.rules[a.ruleEditAt+1:]...)
					} else {
						next = append(next, a.rules...)
						next = append(next, r)
					}
					a.ruleOpen = false
					a.saveRules(next)
				}
			})
		})
	})
}

// ruleTypes are the rule kinds offered in the editor, in mihomo's config
// format. Domain suffix is the most common and the default for new rules.
var ruleTypes = []string{
	"DOMAIN-SUFFIX", "DOMAIN", "DOMAIN-KEYWORD", "DOMAIN-REGEX",
	"GEOSITE", "GEOIP", "IP-CIDR", "IP-CIDR6", "IP-SUFFIX",
	"PROCESS-NAME", "SRC-IP-CIDR", "DST-PORT", "RULE-SET", "MATCH",
}

// ruleTypeConfig turns a rule type, as reported by the API (camelCase) or
// chosen in the editor, into the uppercase hyphenated form mihomo expects.
func ruleTypeConfig(s string) string {
	switch strings.ToUpper(s) {
	case "DOMAINSUFFIX":
		return "DOMAIN-SUFFIX"
	case "DOMAINKEYWORD":
		return "DOMAIN-KEYWORD"
	case "DOMAINREGEX":
		return "DOMAIN-REGEX"
	case "GEOSITE":
		return "GEOSITE"
	case "GEOIP":
		return "GEOIP"
	case "IPCIDR", "IP-CIDR":
		return "IP-CIDR"
	case "IPCIDR6", "IP-CIDR6":
		return "IP-CIDR6"
	case "IPSUFFIX":
		return "IP-SUFFIX"
	case "PROCESSNAME":
		return "PROCESS-NAME"
	case "RULESET":
		return "RULE-SET"
	case "SRCIPCIDR":
		return "SRC-IP-CIDR"
	case "DSTPORT":
		return "DST-PORT"
	}
	return strings.ToUpper(s)
}

// ruleProxies lists the strategies offered in the editor: the built-in
// DIRECT and REJECT, and the proxy groups of the active profile. A rule's
// own strategy is kept even when it isn't one of them, so editing a rule
// never changes what it points to.
func (a *App) ruleProxies() []string {
	seen := map[string]bool{}
	var out []string
	// An empty first option forces the user to pick a strategy explicitly.
	out = append(out, "")
	seen[""] = true
	add := func(s string) {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	add("DIRECT")
	add("REJECT")
	for _, name := range a.groups {
		add(name)
	}
	add(a.ruleProxy)
	return out
}

// defaultPolicyCard sets the strategy of the connections no rule matches.
func (a *App) defaultPolicyCard(c *ui.Context) {
	t := c.Theme()
	current, has := a.defaultPolicy()
	options := a.defaultPolicies()
	choice := current
	if choice == "" {
		choice = options[0]
	}
	card(c).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconShield).FontSize(14).TextColor(t.TextMuted)
			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, "默认策略").FontSize(13).Bold()
				line := "未匹配到任何规则时，连接走 " + choice
				if !has {
					line = "配置里没有 MATCH 规则，未匹配的连接走 " + choice + "（内核默认）"
				}
				ui.Text(c, line).FontSize(11).TextColor(t.TextMuted).SingleLine()
			})
			if ui.Select(c, &choice, options).Width(180).Label("默认策略").Changed() && choice != current {
				a.setDefaultPolicy(choice)
			}
		})
	})
}
