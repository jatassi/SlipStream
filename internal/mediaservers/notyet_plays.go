package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

func (service) Plays(context.Context, link.ServerID, string, link.PlayID) (Plays, error) {
	return Plays{}, ErrNotYet
}
