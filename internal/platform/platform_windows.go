//go:build windows

package platform

import (
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const (
	internetSettings = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	runKey           = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValue         = "clash-go"

	internetOptionRefresh         = 37
	internetOptionSettingsChanged = 39

	bypass = `localhost;127.*;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*;<local>`
)

var (
	wininet               = syscall.NewLazyDLL("wininet.dll")
	internetSetOptionW    = wininet.NewProc("InternetSetOptionW")
	internetOptionPerUser = 0
)

func setSystemProxy(on bool, addr string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettings, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	enable := uint32(0)
	if on {
		enable = 1
	}
	if err := k.SetDWordValue("ProxyEnable", enable); err != nil {
		return err
	}
	if on {
		if err := k.SetStringValue("ProxyServer", addr); err != nil {
			return err
		}
		if err := k.SetStringValue("ProxyOverride", bypass); err != nil {
			return err
		}
	}
	refreshInternetSettings()
	return nil
}

// refreshInternetSettings tells WinINET, and the apps that follow it, that
// the registry's settings changed.
func refreshInternetSettings() {
	for _, option := range []uintptr{internetOptionSettingsChanged, internetOptionRefresh} {
		internetSetOptionW.Call(uintptr(internetOptionPerUser), option, 0, 0)
	}
}

func setAutoStart(on bool, path string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !on {
		if err := k.DeleteValue(runValue); err != nil {
			return err
		}
		return nil
	}
	return k.SetStringValue(runValue, "\""+path+"\"")
}
