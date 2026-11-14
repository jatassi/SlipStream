package mediaservers

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// The reasons a server is unhealthy, as unhealthy_reason stores them.
const (
	reasonNotResponding = "not_responding"
	reasonSendsFailing  = "sends_failing"
	reasonSignInRefused = "sign_in_refused"
	reasonOtherServer   = "other_server"
)

// verdict is a server's health: unhealthy for reason since since, or
// healthy when reason is "".
type verdict struct {
	reason string
	since  time.Time
}

// standing is what judge reads of one server.
type standing struct {
	contact      link.Contact
	contactSince time.Time
	// failingSince is when sends started failing, zero while they succeed.
	failingSince time.Time
	// refusedAt is when plex.tv refused the Plex account's sign-in, zero
	// for a Jellyfin or Emby server.
	refusedAt time.Time
	// up is the reaching Node's link, always up for the Controller.
	up   bool
	held verdict
}

// judge is s's verdict at now, and when it may turn with nothing new
// reported, zero for never. Another server or a refused sign-in turns it
// at once, a server not responding or failing sends once that has lasted
// after. While the reaching Node is away what it last reported is stale,
// so only those it reported at once count and the held verdict stands.
func judge(s standing, now time.Time, after time.Duration) (verdict, time.Time) {
	switch {
	case s.contact == link.ContactOtherServer:
		return verdict{reasonOtherServer, s.contactSince}, time.Time{}
	case s.contact == link.ContactSignInRefused:
		return verdict{reasonSignInRefused, s.contactSince}, time.Time{}
	case !s.refusedAt.IsZero():
		return verdict{reasonSignInRefused, s.refusedAt}, time.Time{}
	case !s.up:
		return s.held, time.Time{}
	}
	var next time.Time
	for _, p := range []struct {
		reason string
		since  time.Time
	}{
		{reasonNotResponding, sinceIf(s.contact == link.ContactNotResponding, s.contactSince)},
		{reasonSendsFailing, s.failingSince},
	} {
		if p.since.IsZero() {
			continue
		}
		due := p.since.Add(after)
		if !now.Before(due) {
			return verdict{p.reason, p.since}, time.Time{}
		}
		next = earlier(next, due)
	}
	return verdict{}, next
}

func sinceIf(ok bool, t time.Time) time.Time {
	if ok {
		return t
	}
	return time.Time{}
}

// settle judges server id now and writes a new verdict, then publishes
// media-servers.unhealthy or media-servers.recovered. It answers when to
// judge again, zero for at the next wake.
func (m *MediaServers) settle(ctx context.Context, id link.ServerID) time.Time {
	var (
		from, to verdict
		kind     link.KindName
		next     time.Time
	)
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		row, err := q.GetServer(ctx, id)
		if err != nil {
			return err
		}
		settings, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		s := standing{
			contact: row.Contact, contactSince: row.ContactSince.Time(), failingSince: row.SendsSince.Time(),
			up: m.up(row.ReachingNodeID), held: heldOf(row),
		}
		if k, _ := m.cfg.Kinds.Of(row.Kind); k.About().Form == link.SignInPlexPIN {
			a, err := q.GetPlexAccount(ctx)
			if err != nil {
				return err
			}
			s.refusedAt = a.RefusedAt.Time()
		}
		from, kind = s.held, row.Kind
		to, next = judge(s, time.Now(), UnhealthyAfter.Value(settings.UnhealthyAfterSeconds))
		if to.reason == from.reason {
			return nil
		}
		return q.SetUnhealthy(ctx, db.SetUnhealthyParams{ID: id, Reason: optional(to.reason), Since: storage.FromTime(to.since)})
	})
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return time.Time{}
	case err != nil:
		if ctx.Err() == nil {
			m.cfg.Log.Error("judge a Media Server's health", "server", id.String(), "err", err.Error())
		}
		return time.Now().Add(m.cfg.Timings.RetryMin)
	case to.reason == from.reason:
	case to.reason == "":
		m.cfg.Log.Info("a Media Server recovered", "server", id.String(), "reason", from.reason)
		m.cfg.Events.Recovered.Publish(ctx, Recovered{ServerID: id.String(), Kind: KindOf(kind), Reason: from.reason})
	default:
		m.cfg.Log.Warn("a Media Server is unhealthy", "server", id.String(), "reason", to.reason)
		m.cfg.Events.Unhealthy.Publish(ctx, WentUnhealthy{ServerID: id.String(), Kind: KindOf(kind), Reason: to.reason})
	}
	return next
}

func heldOf(row db.MediaserversServer) verdict {
	if row.UnhealthyReason == nil {
		return verdict{}
	}
	return verdict{*row.UnhealthyReason, row.UnhealthySince.Time()}
}
