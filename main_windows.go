//go:build windows

package main

import "syscall"

// Hide the console window as early as possible, even when the executable was
// built without -H=windowsgui (for example with plain `go build` or `go run`).
// A proper GUI build has no console, so FreeConsole simply does nothing.
// When launched from an existing terminal this detaches without closing that
// terminal; when double-clicked as a console app it closes the stray window.
func init() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	if p := kernel32.NewProc("FreeConsole"); p.Find() == nil {
		_, _, _ = p.Call()
	}
}
