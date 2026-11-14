package mediaservers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Apply applies one of node's reports inside the receive Write. A report
// from a Node that isn't the server's reaching Node now is dropped with a
// warning, as a re-pointed server's old Node may still send its last
// reports. Only a report that can't be read is refused.
func (m *MediaServers) Apply(ctx context.Context, tx *storage.Tx[storage.Library], node role.NodeID, r *msv1.Report) error {
	id, err := serverIDOf(r.GetServerId())
	if err != nil {
		return err
	}
	q := db.New(tx)
	cur, err := q.GetContact(ctx, id)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return nil
	case err != nil:
		return err
	case cur.ReachingNodeID != node:
		m.cfg.Log.Warn("dropped a Media Server report from a Node that doesn't reach the server",
			"server", id.String(), "from", node.String(), "reaching", cur.ReachingNodeID.String())
		return nil
	}
	switch rep := r.GetReport().(type) {
	case *msv1.Report_Contact:
		c, err := contactOf(rep.Contact)
		if err != nil {
			return err
		}
		if c.version < cur.SettingsVersion {
			return nil
		}
		if c.state != cur.Contact {
			m.senders.wake(id)
		}
		return q.SetContact(ctx, c.next(id, cur))
	case *msv1.Report_PlayStarted:
		return startPlay(ctx, q, id, rep.PlayStarted)
	case *msv1.Report_PlayChanged:
		return changePlay(ctx, q, id, rep.PlayChanged)
	case *msv1.Report_PlayEnded:
		return endPlay(ctx, q, id, rep.PlayEnded)
	case *msv1.Report_Watching:
		return watching(ctx, q, id, rep.Watching)
	case *msv1.Report_IndexHint:
		return hint(ctx, q, id, rep.IndexHint)
	default:
		return fmt.Errorf("a Media Server report of type %T isn't one the Controller applies", rep)
	}
}

func serverIDOf(b []byte) (link.ServerID, error) {
	u, err := storage.UUIDFromBytes(b)
	if err != nil {
		return link.ServerID{}, fmt.Errorf("read the Media Server id: %w", err)
	}
	return link.ServerID{UUID: u}, nil
}

// contact is a Contact report as the Controller reads it.
type contact struct {
	state   link.Contact
	version int64
	at      time.Time
	found   string
}

func contactOf(m *msv1.Contact) (contact, error) {
	state, err := link.ParseContact(m.GetState())
	switch {
	case err != nil:
		return contact{}, err
	case state == link.ContactUnknown:
		return contact{}, errors.New("a Node reported a Media Server's contact as unknown")
	case m.GetAtUnixMs() <= 0:
		return contact{}, errors.New("a Media Server's contact has no time")
	case (state == link.ContactOtherServer) != (m.GetFound() != ""):
		return contact{}, fmt.Errorf("a Media Server contact of %s came with the found identity %q", state, m.GetFound())
	}
	return contact{state: state, version: m.GetSettingsVersion(), at: time.UnixMilli(m.GetAtUnixMs()).UTC(), found: m.GetFound()}, nil
}

// next is the server's contact columns once c applies: since moves only
// when the state changes, and the last contact only while it responds.
func (c contact) next(id link.ServerID, cur db.GetContactRow) db.SetContactParams {
	at := storage.FromTime(c.at)
	p := db.SetContactParams{ID: id, Contact: c.state, ContactSince: cur.ContactSince, LastContactAt: cur.LastContactAt}
	if c.state != cur.Contact {
		p.ContactSince = at
	}
	if c.state == link.ContactResponding {
		p.LastContactAt = at
	}
	if c.found != "" {
		p.ContactFound = &c.found
	}
	return p
}
