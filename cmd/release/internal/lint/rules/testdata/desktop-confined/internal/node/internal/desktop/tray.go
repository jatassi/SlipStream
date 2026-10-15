//go:build windows || darwin

package desktop

import "fyne.io/systray"

var _ = systray.Run
