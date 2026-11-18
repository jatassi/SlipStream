package subtitles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/subtitling"
)

var keyColumn = secret.NewColumn("subtitles_provider.api_key_sealed")

// PauseReason is why a provider takes no calls until a time.
type PauseReason string

const (
	PausedQuota PauseReason = "quota"
	PausedRate  PauseReason = "rate"
	PausedKey   PauseReason = "key"
	PausedDown  PauseReason = "down"
)

// Pause is a provider taking no calls until Until. Words is the provider's
// own message, quoted on screen.
type Pause struct {
	Until  time.Time
	Reason PauseReason
	Words  string
}

// Provider is one Subtitle Provider as the Controller keeps it.
type Provider struct {
	Spec         subtitling.Spec
	Position     int
	On           bool
	HasKey       bool
	Account      string
	Plan         string
	Allowance    *calls.Allowance
	AllowanceAt  time.Time
	Paused       *Pause
	FailingSince time.Time
	LastError    string
	KeyCheckedAt time.Time
}

// Standing is how a provider stands now, the first that holds in this
// order.
type Standing uint8

const (
	Off Standing = iota + 1
	NeedsKey
	Rejected
	Out
	Failing
	Ready
)

// Standing is p's standing at now.
func (p Provider) Standing(now time.Time) Standing {
	switch {
	case !p.On:
		return Off
	case p.Spec.Keyed && !p.HasKey:
		return NeedsKey
	case p.Paused != nil && p.Paused.Reason == PausedKey:
		return Rejected
	case p.Paused != nil && p.Paused.Reason != PausedDown && now.Before(p.Paused.Until):
		return Out
	case !p.FailingSince.IsZero() || p.Paused != nil && p.Paused.Reason == PausedDown && now.Before(p.Paused.Until):
		return Failing
	}
	return Ready
}

// searches is whether p can search: on, and keyed if it needs a key.
func (p Provider) searches() bool { return p.On && (!p.Spec.Keyed || p.HasKey) }

var (
	ErrNoProvider = errors.New("no Subtitle Provider has that id")
	ErrNoHolder   = errors.New("no Node has the Subtitles Role")
	ErrKeyless    = errors.New("the Subtitle Provider takes no key")
)

// OrderMismatchError is an order that doesn't name every provider once.
type OrderMismatchError struct{ Want []subtitling.ProviderID }

func (e *OrderMismatchError) Error() string {
	return fmt.Sprintf("the order must name each of %v once", e.Want)
}

// KeyRefusedError is a key the provider refused, with its own message.
type KeyRefusedError struct {
	Provider string
	Words    string
}

func (e *KeyRefusedError) Error() string { return e.Provider + " refused the key: " + e.Words }

// UnreachableError is a key check that didn't reach the provider, or the
// holder that makes it.
type UnreachableError struct {
	Provider string
	Cause    error
}

func (e *UnreachableError) Error() string {
	return e.Provider + " key check didn't get through: " + e.Cause.Error()
}

func (e *UnreachableError) Unwrap() error { return e.Cause }

func providerOf(spec subtitling.Spec, row db.SubtitlesProvider) (Provider, error) {
	p := Provider{
		Spec: spec, Position: int(row.Position), On: row.IsOn, HasKey: row.ApiKeySealed.Set(), Account: deref(row.Account), Plan: deref(row.Plan),
		AllowanceAt: row.AllowanceAt.Time(), FailingSince: row.FailingSince.Time(), LastError: deref(row.LastError), KeyCheckedAt: row.KeyCheckedAt.Time(),
	}
	if row.PauseReason != nil {
		p.Paused = &Pause{Until: row.PausedUntil.Time(), Reason: PauseReason(*row.PauseReason), Words: deref(row.PauseWords)}
	}
	a, err := allowanceOf(row)
	p.Allowance = a
	return p, err
}

func (s *Subtitles) providersIn(ctx context.Context, r storage.Reader[storage.Library]) ([]Provider, error) {
	rows, err := db.New(r).ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Provider, 0, len(rows))
	for _, row := range rows {
		spec := s.spec(row.ID)
		if spec.ID == "" {
			continue
		}
		p, err := providerOf(spec, row)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// Providers is every Subtitle Provider, in search order.
func (s *Subtitles) Providers(ctx context.Context, _ identity.Admin) ([]Provider, error) {
	var out []Provider
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		out, err = s.providersIn(ctx, r)
		return err
	})
	return out, err
}

func (s *Subtitles) provider(ctx context.Context, id subtitling.ProviderID) (Provider, error) {
	ps, err := s.Providers(ctx, identity.Admin{})
	if err != nil {
		return Provider{}, err
	}
	i := slices.IndexFunc(ps, func(p Provider) bool { return p.Spec.ID == id })
	if i < 0 {
		return Provider{}, ErrNoProvider
	}
	return ps[i], nil
}

// changeProvider runs change on id's provider in one Write, then asks for
// a bulk pass, which makes wants due for a provider that gained the power
// to search.
func (s *Subtitles) changeProvider(ctx context.Context, id subtitling.ProviderID, change func(q *db.Queries, p Provider, now storage.Time) error) (Provider, error) {
	if s.spec(id).ID == "" {
		return Provider{}, ErrNoProvider
	}
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		ps, err := s.providersIn(ctx, tx)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(ps, func(p Provider) bool { return p.Spec.ID == id })
		if i < 0 {
			return ErrNoProvider
		}
		return change(db.New(tx), ps[i], storage.FromTime(s.cfg.Clock()))
	})
	if err != nil {
		return Provider{}, err
	}
	s.providersChanged(ctx)
	return s.provider(ctx, id)
}

func (s *Subtitles) providersChanged(ctx context.Context) {
	s.cfg.ProvidersChanged.Publish(ctx, ProvidersChanged{})
	s.soonReconcile()
}

// SetProviderOn turns id on or off.
func (s *Subtitles) SetProviderOn(ctx context.Context, _ identity.Admin, id subtitling.ProviderID, on bool) (Provider, error) {
	return s.changeProvider(ctx, id, func(q *db.Queries, p Provider, now storage.Time) error {
		params := db.SetProviderOnParams{ID: id, IsOn: on}
		was := p
		p.On = on
		if p.searches() && !was.searches() {
			params.GainedAt = now
		}
		return q.SetProviderOn(ctx, params)
	})
}

// OrderProviders sets the search order to ids, which names every provider
// once.
func (s *Subtitles) OrderProviders(ctx context.Context, _ identity.Admin, ids []subtitling.ProviderID) ([]Provider, error) {
	all := make([]subtitling.ProviderID, len(s.cfg.Providers))
	for i, p := range s.cfg.Providers {
		all[i] = p.ID
	}
	sorted := slices.Sorted(slices.Values(ids))
	if !slices.Equal(sorted, slices.Sorted(slices.Values(all))) {
		return nil, &OrderMismatchError{Want: all}
	}
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		if err := q.ShiftPositions(ctx, int64(len(ids))); err != nil {
			return err
		}
		for i, id := range ids {
			if err := q.SetPosition(ctx, db.SetPositionParams{ID: id, Position: int64(i + 1)}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.cfg.ProvidersChanged.Publish(ctx, ProvidersChanged{})
	return s.Providers(ctx, identity.Admin{})
}

// SetKey checks key on a Subtitles holder, then seals and stores it with
// the account, plan and allowance the check answered. A refused key is
// *KeyRefusedError with the provider's own words, and nothing is stored.
func (s *Subtitles) SetKey(ctx context.Context, _ identity.Admin, id subtitling.ProviderID, key string) (Provider, error) {
	spec := s.spec(id)
	switch {
	case spec.ID == "":
		return Provider{}, ErrNoProvider
	case !spec.Keyed:
		return Provider{}, ErrKeyless
	}
	holders := s.cfg.Roster.Holding(role.Subtitles)
	if len(holders) == 0 {
		return Provider{}, ErrNoHolder
	}
	account, outcomes, err := s.cfg.Holders(holders[0]).CheckKey(ctx, id, key)
	if err != nil {
		return Provider{}, &UnreachableError{Provider: spec.Label, Cause: err}
	}
	allowance, at, err := checked(spec, outcomes)
	if err != nil {
		return Provider{}, err
	}
	var stored *string
	if allowance != nil {
		b, err := json.Marshal(allowance)
		if err != nil {
			return Provider{}, fmt.Errorf("subtitles: %s's allowance: %w", spec.Label, err)
		}
		stored = new(string(b))
	}
	return s.changeProvider(ctx, id, func(q *db.Queries, _ Provider, now storage.Time) error {
		_, err := q.PutProviderKey(ctx, db.PutProviderKeyParams{
			ID: id, ApiKeySealed: s.cfg.Sealer.Seal([]byte(key), keyColumn.Row(string(id))), Account: some(account.Name), Plan: some(account.Plan),
			Allowance: stored, AllowanceAt: at, Now: now,
		})
		return err
	})
}

// checked is the allowance a key check on spec reported, or why the key
// isn't stored. A provider that is only out of downloads took the key.
func checked(spec subtitling.Spec, outcomes []calls.Outcome) (*calls.Allowance, storage.Time, error) {
	var allowance *calls.Allowance
	var at storage.Time
	for _, o := range outcomes {
		if o.Provider != spec.ID {
			continue
		}
		switch o.Kind {
		case calls.Refused, calls.PinRefused:
			return nil, storage.Time{}, &KeyRefusedError{Provider: spec.Label, Words: o.Words}
		case calls.Unreachable, calls.Limited, calls.NotFound:
			return nil, storage.Time{}, &UnreachableError{Provider: spec.Label, Cause: errors.New(o.Words)}
		case calls.OK, calls.Capped:
		}
		if o.Allowance != nil {
			allowance, at = o.Allowance, storage.FromTime(o.At)
		}
	}
	return allowance, at, nil
}

// DeleteKey forgets id's key and what its checks reported.
func (s *Subtitles) DeleteKey(ctx context.Context, _ identity.Admin, id subtitling.ProviderID) (Provider, error) {
	if s.spec(id).ID != "" && !s.spec(id).Keyed {
		return Provider{}, ErrKeyless
	}
	return s.changeProvider(ctx, id, func(q *db.Queries, _ Provider, _ storage.Time) error {
		_, err := q.DeleteProviderKey(ctx, id)
		return err
	})
}
