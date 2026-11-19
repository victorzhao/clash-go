package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/victorzhao/clash-go/internal/mihomoapi"
)

func (a *App) connectionsPage(c *ui.Context) {
	t := c.Theme()
	if !a.running.Load() {
		a.emptyState(c, "请求", "内核未运行，启动后才能看到请求。", true)
		return
	}
	filtered := a.filteredConnections()
	ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
			a.pageTitle(c, "请求", fmt.Sprintf("%d 条请求", len(filtered)))
			ui.Spacer(c)
			ui.SearchField(c, &a.connFilter).Width(160).Placeholder("筛选主机")
			if ui.Button(c, "关闭全部").Clicked() {
				a.closeAllConnections()
			}
		})

		cols := []ui.TableColumn{
			{Title: "主机", ID: "host", Width: 210},
			{Title: "类型", Width: 85},
			{Title: "规则", Width: 130},
			{Title: "链路", Width: 150},
			{Title: "下载", Width: 75, Align: ui.End},
			{Title: "上传", Width: 75, Align: ui.End},
			{Title: "时长", Width: 80, Align: ui.End},
		}
		ui.Column(c).Grow(1).Radius(16).Background(cardColor(c)).
			Border(1, cardEdge(c)).Shadow(0, 2, 8, 0, cardShadow(c)).Padding(6).Children(func() {
			ui.Table(c, &a.connList, cols, len(filtered), func(row, col int) {
				conn := filtered[row]
				switch col {
				case 0:
					host := conn.Metadata.Host
					if host == "" {
						host = conn.Metadata.Destination
					}
					ui.Text(c, host).SingleLine().Tooltip(fmt.Sprintf("%s:%s", host, conn.Metadata.DstPort))
				case 1:
					ui.Text(c, conn.Metadata.Type+" "+conn.Metadata.Network).TextColor(t.TextMuted).SingleLine()
				case 2:
					ui.Text(c, conn.Rule).TextColor(t.TextMuted).SingleLine().Tooltip(conn.Payload)
				case 3:
					chain := strings.Join(conn.Chains, " → ")
					ui.Text(c, chain).TextColor(t.TextMuted).SingleLine()
				case 4:
					ui.Text(c, formatBytes(conn.Download)).FontFeatures("tnum").SingleLine()
				case 5:
					ui.Text(c, formatBytes(conn.Upload)).FontFeatures("tnum").SingleLine()
				case 6:
					ui.Text(c, formatDuration(time.Since(conn.Start))).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine()
				}
			}).Grow(1).Padding(0, 0, 0, 0)
		})
	})
}

func (a *App) filteredConnections() []mihomoapi.Connection {
	filter := strings.ToLower(strings.TrimSpace(a.connFilter))
	if filter == "" {
		return a.conns
	}
	var out []mihomoapi.Connection
	for _, c := range a.conns {
		if strings.Contains(strings.ToLower(c.Metadata.Host), filter) ||
			strings.Contains(strings.ToLower(c.Metadata.Destination), filter) ||
			strings.Contains(strings.ToLower(strings.Join(c.Chains, " ")), filter) {
			out = append(out, c)
		}
	}
	return out
}

func (a *App) closeAllConnections() {
	go func() {
		err := a.api.CloseAllConnections()
		a.update(func() {
			if err != nil {
				a.toastText = "关闭失败：" + err.Error()
				return
			}
			a.conns = nil
			a.connCount = 0
			a.toastText = "已关闭全部连接"
		})
	}()
}
