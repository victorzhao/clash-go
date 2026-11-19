// Clash Go is a Clash (mihomo) desktop client in Go, its interface drawn by
// MyGo's native UI toolkit: dashboard, proxies, profiles, connections, rules,
// logs and settings, with the core running in the app's own process.
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/victorzhao/clash-go/internal/app"
)

// doubleClickWindowMS is the longest gap between two left clicks on the
// tray icon that still counts as a double click, matching Windows' own
// double-click timing closely enough for a snappy feel.
const doubleClickWindowMS = 500

func main() {
	// Only one instance runs. A later launch hands its command line over to
	// the first one, which shows its window in OnSecondInstance, then exits.
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}

	a, err := app.New()
	if err != nil {
		log.Fatal(err)
	}
	var win *mygo.Window
	// A second launch during startup is remembered and handled once the
	// window exists (OnSecondInstance runs on the main thread, like WhenReady).
	var showWhenReady bool
	showWindow := func() {
		if win != nil {
			win.Show()
			win.Focus()
		} else {
			showWhenReady = true
		}
	}
	mygo.App.OnSecondInstance(func([]string, string) { showWindow() })
	// Set when the app begins quitting, so the window's OnClose handler
	// knows to let the window close instead of hiding it. Without this,
	// closing the window to hide it cancels the tray's Quit.
	var quitting bool

	mygo.App.WhenReady(func() {
		win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "Clash Go",
			Width:     1500,
			Height:    1000,
			MinWidth:  1100,
			MinHeight: 760,
			StateKey:  "main",
			Content:   ui.View(a.View),
		})
		// Give the window its own icon in the title bar and taskbar. A plain
		// go build does not embed an executable icon, so set it at runtime.
		if err := win.SetIcon(app.IconPNG(256)); err != nil {
			log.Println("window icon:", err)
		}
		a.Attach(win)
		// Closing the window hides it: the app keeps running in the tray,
		// and quits from the tray's menu. During a quit, let it close so
		// App.Quit is not cancelled.
		win.OnClose(func(e *mygo.CloseEvent) {
			if quitting {
				return
			}
			e.PreventDefault()
			win.Hide()
		})

		tray, err := mygo.NewTray(mygo.TrayOptions{
			// The tray shows the same blue blossom as the app icon, not a
			// tinted template, so the two always look identical.
			Icon:           app.IconPNG(32),
			IconIsTemplate: false,
			ToolTip:        "Clash Go  ·  双击显示窗口，右键打开菜单",
		})
		if err != nil {
			log.Println("tray:", err)
		} else {
			// No menu is installed on the icon: a left click reaches OnClick
			// (where a double click shows the window) and a right click opens
			// the menu above the icon (wired in App.SetTray).
			var lastClick time.Time
			tray.OnClick(func() {
				now := time.Now()
				if now.Sub(lastClick) <= doubleClickWindowMS*time.Millisecond {
					showWindow()
					lastClick = time.Time{}
					return
				}
				lastClick = now
			})
			a.SetTray(tray)
		}
		if showWhenReady {
			showWhenReady = false
			showWindow()
		}
	})

	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		quitting = true
		a.Shutdown()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
