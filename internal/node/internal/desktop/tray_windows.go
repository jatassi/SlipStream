package desktop

import (
	_ "embed"

	"fyne.io/systray"
)

//go:embed tray.ico
var icon []byte

func (t *tray) osReady() {
	systray.SetIcon(icon)
	if err := watchSessionEnd(t.app); err != nil {
		t.log.Warn("watch for the session end", "err", err.Error())
	}
}
