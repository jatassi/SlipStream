//go:build !windows

package desktop

import (
	"fmt"

	"fyne.io/systray/internal/generated/menu"
)

var _, _ = fmt.Sprint, menu.X
