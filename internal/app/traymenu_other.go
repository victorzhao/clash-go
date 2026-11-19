//go:build !windows

package app

import "github.com/egoist/mygo"

// showTrayMenu falls back to MyGo's own menu on platforms where the custom
// above-the-icon popup is not implemented. The menu is set only for the
// duration of the popup so primary clicks remain free for double-click.
func showTrayMenu(tray *mygo.Tray, items []trayMenuItem, hwnd uintptr) {
	if tray == nil {
		return
	}
	tray.SetMenu(buildMygoMenu(items))
	tray.PopUpMenu()
	tray.SetMenu(nil)
}

func buildMygoMenu(items []trayMenuItem) *mygo.Menu {
	mi := make([]*mygo.MenuItem, 0, len(items))
	for _, it := range items {
		switch {
		case it.sep:
			mi = append(mi, mygo.Separator())
		case len(it.submenu) > 0:
			mi = append(mi, &mygo.MenuItem{Label: it.label, Submenu: buildMygoItems(it.submenu)})
		default:
			item := &mygo.MenuItem{Label: it.label, Checked: it.checked}
			if it.onClick != nil {
				item.Click = func(*mygo.MenuItem, *mygo.Window) { it.onClick() }
			}
			mi = append(mi, item)
		}
	}
	return mygo.NewMenu(mi)
}

func buildMygoItems(items []trayMenuItem) []*mygo.MenuItem {
	mi := make([]*mygo.MenuItem, 0, len(items))
	for _, it := range items {
		switch {
		case it.sep:
			mi = append(mi, mygo.Separator())
		case len(it.submenu) > 0:
			mi = append(mi, &mygo.MenuItem{Label: it.label, Submenu: buildMygoItems(it.submenu)})
		default:
			item := &mygo.MenuItem{Label: it.label, Checked: it.checked}
			if it.onClick != nil {
				item.Click = func(*mygo.MenuItem, *mygo.Window) { it.onClick() }
			}
			mi = append(mi, item)
		}
	}
	return mi
}
