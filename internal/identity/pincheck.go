package identity

import (
	"context"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/mail"
)

// subject is whose PIN a check is for: an Account, or an email with none,
// which verifies against the dummy hash.
type subject struct {
	account AccountID
	email   string
	name    string
	hash    string
}

func subjectOf(row db.IdentityAccount) subject {
	return subject{account: row.ID, email: row.Email, name: row.DisplayName, hash: row.PinHash}
}

func (s subject) known() bool { return !s.account.IsZero() }

func (s subject) mailbox() mail.Address { return mail.Address{Name: s.name, Email: s.email} }

// pinPass proves a PIN checked out. Only checkPIN builds one and only
// afterPIN consumes it.
type pinPass struct {
	account AccountID
	r       reservation
}

// checkPIN is the one PIN check, for sign-in, re-confirm and Change PIN. An
// address over its limit is refused first. The attempt is then reserved,
// before it queues for a hash slot, so however many run at once no more
// than the hard stop's number are ever verified.
func (s *Service) checkPIN(ctx context.Context, c client, sub subject, pin PIN, set Settings) (pinPass, error) {
	if err := s.admit(c, set); err != nil {
		return pinPass{}, err
	}
	r, err := s.reserve(ctx, sub, limitsOf(set), time.Now())
	if err != nil {
		return pinPass{}, err
	}
	ok, err := s.pins.verify(ctx, pin, sub.hash)
	if err != nil {
		return pinPass{}, err
	}
	if ok && sub.known() {
		return pinPass{account: sub.account, r: r}, nil
	}
	s.fail(c, set)
	if r.reachedOff && sub.known() {
		if err := post(s, sub.mailbox(), pinOffMail, pinOffData{Email: sub.email, Count: r.n}); err != nil {
			return pinPass{}, err
		}
	}
	return pinPass{}, ErrPINWrong
}

func (s *Service) admit(c client, set Settings) error {
	if until, blocked := s.limiter.blocked(c.address, set.AddressFailures, set.AddressWindow, time.Now()); blocked {
		return &WaitError{Kind: TooManyAttempts, Until: until}
	}
	return nil
}

func (s *Service) fail(c client, set Settings) {
	s.limiter.fail(c.address, set.AddressFailures, set.AddressWindow, time.Now())
}

func (s *Service) reserve(ctx context.Context, sub subject, lim limits, now time.Time) (reservation, error) {
	if !sub.known() {
		return s.shadow.reserve(sub.email, lim, now)
	}
	var r reservation
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, _ *effects) error {
		row, err := q.GetAccount(ctx, sub.account)
		if err != nil {
			return err
		}
		var l ledger
		if l, r, err = reserve(ledgerOf(row), lim, now); err != nil {
			return err
		}
		return q.SavePINLedger(ctx, l.save(sub.account))
	})
	return r, err
}

// afterPIN runs a right PIN's Write: settle the ledger, then fn. fn failing
// rolls the settle back too.
func (s *Service) afterPIN(ctx context.Context, p pinPass, fn func(ctx context.Context, q *db.Queries, fx *effects) error) error {
	return s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		row, err := q.GetAccount(ctx, p.account)
		if err != nil {
			return err
		}
		l, err := settle(ledgerOf(row), p.r)
		if err != nil {
			return err
		}
		if err := q.SavePINLedger(ctx, l.save(p.account)); err != nil {
			return err
		}
		return fn(ctx, q, fx)
	})
}

// shadowLedger holds the ledgers of emails with no Account, the least
// recently touched forgotten past shadowCap, and none after a restart. A
// prober can still tell an Account exists by outlasting an entry; the
// per-address limit makes that slow and a real owner is emailed.
type shadowLedger struct {
	mu      sync.Mutex
	entries map[string]*shadowEntry
	touches uint64
}

type shadowEntry struct {
	l       ledger
	touched uint64
}

const shadowCap = 4096

func newShadowLedger() *shadowLedger {
	return &shadowLedger{entries: map[string]*shadowEntry{}}
}

func (sh *shadowLedger) reserve(email string, lim limits, now time.Time) (reservation, error) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	e, ok := sh.entries[email]
	if !ok {
		if len(sh.entries) >= shadowCap {
			sh.forgetStalest()
		}
		e = &shadowEntry{}
		sh.entries[email] = e
	}
	sh.touches++
	e.touched = sh.touches
	l, r, err := reserve(e.l, lim, now)
	e.l = l
	return r, err
}

func (sh *shadowLedger) forgetStalest() {
	var stalest string
	var at uint64
	for email, e := range sh.entries {
		if at == 0 || e.touched < at {
			stalest, at = email, e.touched
		}
	}
	delete(sh.entries, stalest)
}
