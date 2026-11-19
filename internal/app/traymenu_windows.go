//go:build windows

package app

import (
	"sync"
	"syscall"
	"unsafe"

	"github.com/egoist/mygo"
)

// Win32 constants used to build and place the tray's popup menu. These
// mirror the values MyGo's own Windows menu backend uses.
const (
	mfChecked   = 0x0008
	mfPopup     = 0x0010
	mfSeparator = 0x0800
	mfString    = 0x0000

	tpmReturnCmd   = 0x0000 // TrackPopupMenuEx returns the chosen item's id
	tpmRightButton = 0x0002
	tpmBottomAlign = 0x0020
	tpmCenterAlign = 0x0100
	tpmVertical    = 0x0004
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")

	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procTrackPopupMenuEx    = user32.NewProc("TrackPopupMenuEx")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procGetDpiForSystem     = user32.NewProc("GetDpiForSystem")
)

// toUTF16 converts a Go string to a NUL-terminated UTF-16 buffer for Win32.
func toUTF16(s string) *uint16 {
	b, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return nil
	}
	return b
}

// commandTable maps native menu item ids back to their Go callbacks. It is
// rebuilt on each popup, so stale ids from a dismissed menu can never fire.
var (
	cmdMu    sync.RWMutex
	cmdTable = map[uint16]func(){}
	cmdNext  = 1
)

func registerCommand(fn func()) uint16 {
	cmdMu.Lock()
	defer cmdMu.Unlock()
	id := uint16(cmdNext)
	cmdNext++
	if cmdNext > 0xFFFE {
		cmdNext = 1
	}
	cmdTable[id] = fn
	return id
}

// buildNativeMenu recursively creates a Win32 HMENU from items. Checked
// items are drawn with MF_CHECKED; a submenu becomes an MF_POPUP row whose
// handle is the recursively built child menu.
func buildNativeMenu(items []trayMenuItem) uintptr {
	h, _, _ := procCreatePopupMenu.Call()
	if h == 0 {
		return 0
	}
	for _, it := range items {
		switch {
		case it.sep:
			procAppendMenuW.Call(h, mfSeparator, 0, 0)
		case len(it.submenu) > 0:
			sub := buildNativeMenu(it.submenu)
			procAppendMenuW.Call(h, mfPopup|mfString, sub, uintptr(unsafe.Pointer(toUTF16(it.label))))
		default:
			flags := uintptr(mfString)
			if it.checked {
				flags |= mfChecked
			}
			id := uintptr(0)
			if it.onClick != nil {
				id = uintptr(registerCommand(it.onClick))
			}
			procAppendMenuW.Call(h, flags, id, uintptr(unsafe.Pointer(toUTF16(it.label))))
		}
	}
	return h
}

// systemDPI returns the system DPI, falling back to 96 on older Windows.
func systemDPI() uint32 {
	if procGetDpiForSystem.Find() != nil {
		return 96
	}
	dpi, _, _ := procGetDpiForSystem.Call()
	if dpi == 0 {
		return 96
	}
	return uint32(dpi)
}

// showTrayMenu builds items as a native popup placed just above the tray
// icon and blocks until a choice is made. hwnd is a window handle used as
// the menu's owner.
func showTrayMenu(tray *mygo.Tray, items []trayMenuItem, hwnd uintptr) {
	// Drop callbacks from an earlier, dismissed popup.
	cmdMu.Lock()
	cmdTable = map[uint16]func(){}
	cmdMu.Unlock()

	h := buildNativeMenu(items)
	if h == 0 {
		return
	}
	defer procDestroyMenu.Call(h)

	// Tray.Bounds is in DIPs; TrackPopupMenuEx takes physical pixels.
	bounds := tray.Bounds()
	dpi := systemDPI()
	scale := float64(dpi) / 96.0
	cx := int32(float64(bounds.X+bounds.Width/2) * scale)
	top := int32(float64(bounds.Y) * scale)
	gap := int32(12 * scale) // same gap MyGo uses above the taskbar

	// The owner must be in the foreground or the menu will not close when
	// the user clicks elsewhere.
	if hwnd != 0 {
		procSetForegroundWindow.Call(hwnd)
	}

	// Bottom-align the menu at (cx, top-gap) and center it on the icon, so
	// it always opens above the icon regardless of the taskbar's edge.
	cmd, _, _ := procTrackPopupMenuEx.Call(
		h,
		tpmReturnCmd|tpmRightButton|tpmCenterAlign|tpmBottomAlign|tpmVertical,
		uintptr(cx), uintptr(top-gap), hwnd, 0,
	)
	if hwnd != 0 {
		procPostMessageW.Call(hwnd, 0, 0, 0) // WM_NULL, as Win32 requires
	}

	cmdMu.RLock()
	fn := cmdTable[uint16(cmd)]
	cmdMu.RUnlock()
	if fn != nil {
		fn()
	}
}
