package desktop

import (
	_ "embed"

	"fyne.io/systray"
)

// icon is a template image, which AppKit draws in the menu bar's own color.
//
//go:embed tray-template.png
var icon []byte

func (t *tray) osReady() { systray.SetTemplateIcon(icon, icon) }

// RestartOnMainThread is true because AppKit owns the main thread and
// exits the process once the tray's exit hook returns, so a restart's exec
// runs in that hook.
const RestartOnMainThread = true
