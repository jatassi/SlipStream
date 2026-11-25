package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/mediaservers/access"
)

// actLocked runs on on r for a caller that holds r's turn: every Admin
// and Member operation on one row. An event already past answers the row
// as it is, so the second of two Admins succeeds.
func (m *MediaServers) actLocked(ctx context.Context, r accessRow, on access.Event, run runStep) (Acted, error) {
	if !access.Starts(r.State, on, r.Way) {
		if access.Past(r.State, on, r.Way) {
			return m.acted(ctx, r.ID)
		}
		return Acted{}, &AccessStateError{Member: r.MemberName, Server: r.server.Name}
	}
	st, err := run(ctx, r)
	if err != nil {
		return Acted{}, err
	}
	t, err := access.Next(r.State, on, st.outcome, r.Way)
	if err != nil {
		return Acted{}, err
	}
	w, err := m.move(ctx, r, t, st)
	if err != nil {
		return Acted{}, err
	}
	m.afterMove(ctx, w)
	if t.To == access.None {
		return Acted{}, nil
	}
	return m.acted(ctx, r.ID)
}

// acted is row id as the Admin reads it after an operation.
func (m *MediaServers) acted(ctx context.Context, id AccessID) (Acted, error) {
	v, err := m.view(ctx, id)
	if err != nil {
		return Acted{}, err
	}
	return Acted{Row: &v}, nil
}
