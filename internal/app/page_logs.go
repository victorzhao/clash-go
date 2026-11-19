package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// logsText renders the current log buffer as plain text, one line per entry,
// for copying to the clipboard with the "复制" button.
func (a *App) logsText() string {
	var b strings.Builder
	for _, e := range a.logs {
		b.WriteString(e.At.Format("15:04:05"))
		b.WriteByte(' ')
		b.WriteString(levelName(e.Level))
		b.WriteByte(' ')
		b.WriteString(e.Message)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func levelName(level string) string {
	switch strings.ToLower(level) {
	case "debug":
		return "DEBUG"
	case "warning", "warn":
		return "WARN"
	case "error":
		return "ERROR"
	case "info":
		return "INFO"
	}
	return strings.ToUpper(level)
}

func levelColor(t *ui.Theme, level string) ui.Color {
	switch strings.ToLower(level) {
	case "error":
		return t.Danger
	case "warning", "warn":
		return t.Warning
	case "debug":
		return t.TextMuted
	}
	return t.Text
}

func (a *App) logsPage(c *ui.Context) {
	t := c.Theme()
	a.logList.FollowEnd = true
	ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
			a.pageTitle(c, "日志", "内核输出的日志，最新在底部")
			ui.Spacer(c)
			if a.logPause {
				if ui.Button(c, "继续").Clicked() {
					a.logPause = false
				}
			} else if ui.Button(c, "暂停").Clicked() {
				a.logPause = true
			}
			if ui.Button(c, "复制").Clicked() {
				if text := a.logsText(); text != "" {
					c.WriteClipboard(text)
					a.toastText = "已复制日志"
				} else {
					a.toastText = "还没有日志可复制"
				}
			}
			if ui.Button(c, "清空").Clicked() {
				a.logs = nil
			}
		})

		box := ui.Column(c).Grow(1).Radius(14).Background(cardColor(c)).
			Border(1, cardEdge(c)).Padding(8)
		box.Children(func() {
			// Selectable makes all rows share one selection scope: drag to
			// select, double-click a word, Cmd/Ctrl+C to copy, and a built-in
			// right-click "Copy / Select All" menu.
			ui.List(c, &a.logList, len(a.logs), func(i int) {
				entry := a.logs[i]
				ui.Row(c).Gap(8).Padding(3, 6).Children(func() {
					ui.Text(c, entry.At.Format("15:04:05")).FontSize(11).
						TextColor(t.TextMuted).FontFeatures("tnum").NoWrap()
					ui.Text(c, levelName(entry.Level)).FontSize(11).Bold().
						TextColor(levelColor(t, entry.Level)).Width(46).NoWrap()
					ui.Text(c, entry.Message).FontSize(12).TextColor(levelColor(t, entry.Level)).Grow(1)
				})
			}).Grow(1).Selectable().Children(func() {
				if len(a.logs) == 0 {
					ui.Text(c, "还没有日志").FontSize(12).TextColor(t.TextMuted).Padding(8)
				}
			})
		})
	})
}
