// Package platform changes the system's settings the app needs: the
// system proxy and starting the app when the user signs in. Only Windows
// implements them; elsewhere they do nothing.
package platform

// SetSystemProxy points the system's proxy at addr ("127.0.0.1:7890") when
// on is true, and turns it off when false. It returns whether the system
// took the change.
func SetSystemProxy(on bool, addr string) error { return setSystemProxy(on, addr) }

// SetAutoStart adds the executable at path to the programs that start when
// the user signs in, or takes it out.
func SetAutoStart(on bool, path string) error { return setAutoStart(on, path) }
