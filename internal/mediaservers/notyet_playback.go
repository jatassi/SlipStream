package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/playback"
)

func (service) Playback(context.Context, link.ServerID) (playback.Report, error) {
	return playback.Report{}, ErrNotYet
}
