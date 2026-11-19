//go:build !windows

package platform

func setSystemProxy(on bool, addr string) error { return nil }

func setAutoStart(on bool, path string) error { return nil }
