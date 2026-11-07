//go:build !windows

package app

type desktopShell struct{}

func (a *app) initDesktop()  {}
func (a *app) syncDesktop()  {}
func (a *app) closeDesktop() {}
