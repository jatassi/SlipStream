package mediaservers

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/providers/plextv"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// renewalUnits is the signed-in account, so the task is listed only while
// SlipStream is signed in to Plex.
func (m *MediaServers) renewalUnits(ctx context.Context, r storage.Reader[storage.Library]) ([]tasks.Unit, error) {
	a, ok, err := m.readAccount(ctx, db.New(r))
	if err != nil || !ok || !a.signedInNow() {
		return nil, err
	}
	return []tasks.Unit{{Key: "plex-account", Label: a.name, Place: tasks.OnController()}}, nil
}

// renew renews the Plex JWT when less than the margin, or one cadence and
// a call, is left (J1), then reads the servers again. A refusal marks the
// account refused, and its every Plex server reads it (J4).
func (m *MediaServers) renew(ctx context.Context, _ tasks.Unit, _ *tasks.Progress) error {
	a, err := m.signedIn(ctx)
	if errors.Is(err, ErrPlexNotSignedIn) {
		return nil
	}
	if err != nil {
		return err
	}
	threshold, err := m.renewWhenLeft(ctx)
	if err != nil {
		return err
	}
	if time.Until(a.expires) < threshold {
		token, err := m.plexTV(a).Renew(ctx)
		if plextv.Refused(err) {
			m.refuse(ctx)
		}
		if err != nil {
			return renewal(fromPlexTV(err))
		}
		err = m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return db.New(tx).RenewPlex(ctx, db.RenewPlexParams{
				JwtSealed: m.cfg.Sealer.Seal([]byte(token.JWT), jwtColumn.Single()), JwtExpiresAt: storage.FromTime(token.Expires), RenewedAt: storage.Now(),
			})
		})
		if err != nil {
			return err
		}
		m.cfg.Log.Info("renewed the Plex sign-in", "expires", token.Expires)
		m.senders.wakeAll()
	}
	return renewal(m.reread(ctx))
}

// renewal is a refused sign-in as an owned skip, since the Media Servers'
// own state names it.
func renewal(err error) error {
	if refused, ok := errors.AsType[*SignInRefusedError](err); ok {
		return tasks.SkipOwned(refusedWords(refused))
	}
	return err
}

// renewWhenLeft is max(margin, cadence + call timeout), so a sign-in never
// expires between two runs.
func (m *MediaServers) renewWhenLeft(ctx context.Context) (time.Duration, error) {
	every, err := m.cfg.Tasks.Every(ctx, RenewalTaskID)
	if err != nil {
		return 0, err
	}
	var row db.MediaserversSetting
	err = m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		row, err = db.New(r).GetSettings(ctx)
		return err
	})
	return max(RenewMargin.Value(row.RenewMarginSeconds), RenewEvery.Value(every)+m.cfg.Timings.Call), err
}
