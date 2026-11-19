// Package app — tray menu model.
//
// The tray's left click shows the main window (on a double click) and its
// right click opens this menu. The menu is a small, platform-independent
// description; traymenu_windows.go turns it into a native popup placed just
// above the tray icon, and traymenu_other.go falls back to MyGo's menu on
// platforms where the custom popup is not available.
package app

// trayMenuItem describes one row of the tray's right-click menu.
type trayMenuItem struct {
	label   string         // text shown for the item
	checked bool           // whether a check/radio mark is shown
	radio   bool           // draw the mark as a radio dot rather than a tick
	onClick func()         // called when the item is chosen
	submenu []trayMenuItem // nested items; a non-empty slice makes a submenu
	sep     bool           // this item is a separator
}

// sepItem is the separator row used between groups of tray menu items.
func sepItem() trayMenuItem { return trayMenuItem{sep: true} }
