package identity

import (
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// ledger is the PIN-failure state of one subject: an Account's row, or a
// shadow entry for an email with no Account. reserve and settle are the
// whole state machine, and both stores apply them unchanged, so an unknown
// email answers exactly as a real one.
type ledger struct {
	failures  int
	waitUntil time.Time
	offAt     time.Time
}

type limits struct {
	waitAfter, offAfter int
	wait                time.Duration
}

func limitsOf(s Settings) limits {
	return limits{waitAfter: s.PINWaitAfter, offAfter: s.PINOffAfter, wait: s.PINWait}
}

// reservation is one attempt, counted before its hash runs.
type reservation struct {
	// n is the failure count this attempt made, unique per subject.
	n          int
	reachedOff bool
}

type WaitKind uint8

const (
	PINWait WaitKind = iota
	TooManyAttempts
)

type WaitError struct {
	Kind  WaitKind
	Until time.Time
}

func (e *WaitError) Error() string { return "identity: try again at " + e.Until.Format(time.RFC3339) }

var ErrPINOff = errors.New("identity: the PIN is turned off")

// reserve is a PIN check's first step. Off refuses, and a wait refuses
// until it passes, counting nothing. Otherwise the attempt counts: reaching
// waitAfter, or any count past it, starts the wait again from now, and
// reaching offAfter turns the PIN off.
func reserve(l ledger, lim limits, now time.Time) (ledger, reservation, error) {
	switch {
	case !l.offAt.IsZero():
		return l, reservation{}, ErrPINOff
	case now.Before(l.waitUntil):
		return l, reservation{}, &WaitError{Kind: PINWait, Until: l.waitUntil}
	}
	l.failures++
	if l.failures >= lim.waitAfter {
		l.waitUntil = now.Add(lim.wait)
	}
	if l.failures >= lim.offAfter {
		l.offAt = now
	}
	return l, reservation{n: l.failures, reachedOff: !l.offAt.IsZero()}, nil
}

// settle is a right PIN's second step, clearing the ledger. When an attempt
// reserved after r turned the PIN off, its verdict stands.
func settle(l ledger, r reservation) (ledger, error) {
	if !l.offAt.IsZero() && l.failures != r.n {
		return l, ErrPINOff
	}
	return ledger{}, nil
}

func ledgerOf(row db.IdentityAccount) ledger {
	return ledger{failures: int(row.PinFailures), waitUntil: row.PinWaitUntil.Time(), offAt: row.PinDisabledAt.Time()}
}

func (l ledger) save(id AccountID) db.SavePINLedgerParams {
	return db.SavePINLedgerParams{
		PinFailures:   int64(l.failures),
		PinWaitUntil:  storage.FromTime(l.waitUntil),
		PinDisabledAt: storage.FromTime(l.offAt),
		ID:            id,
	}
}
