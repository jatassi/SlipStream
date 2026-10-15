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
