package mediaservers

import (
	"time"

	"github.com/jatassi/slipstream/internal/mediaservers/playback"
)

// window is the plays a server's history keeps at now: those that started
// after the later of now minus keep and playsSince.
func window(now time.Time, keep time.Duration, playsSince time.Time) playback.Window {
	return playback.Window{Since: later(now.Add(-keep), playsSince), Keep: keep}
}
