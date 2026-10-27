package indexers

import (
	"context"
	"errors"
	"maps"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	// ErrCaptchaExpired is an answer to a captcha attempt that expired, was
	// answered before, or was shown for another Node or username.
	ErrCaptchaExpired = errors.New("the captcha expired")
	// ErrNoCaptcha is a captcha asked for an Indexer whose sign-in shows
	// none.
	ErrNoCaptcha = errors.New("the Indexer's sign-in shows no captcha")
)

// CaptchaAnswer is the Admin's Text for the captcha of Attempt.
type CaptchaAnswer struct{ Attempt, Text string }

// Captcha is a captcha for the Admin to answer, open as Attempt until
// Expires.
type Captcha struct {
	Attempt     string
	Image       []byte
	ContentType string
	Expires     time.Time
}

// CaptchaFor is the Indexer a captcha is for: Draft, or else the saved
// Indexer.
type CaptchaFor struct {
	Draft   *Draft
	Indexer ID
}

// attempts are the captchas shown and not yet answered, by attempt id. The
// site's pre-login state lives only here, in the Controller's memory, and
// only until its attempt expires.
type attempts struct {
	mu   sync.Mutex
	byID map[string]attempt
}

// attempt is a captcha shown for a sign-in on node as username to a site
// of definition.
type attempt struct {
	node                 role.NodeID
	definition, username string
	preLogin             []byte
	expires              time.Time
}

func (as *attempts) keep(a attempt) string {
	id := storage.NewUUID().String()
	as.mu.Lock()
	defer as.mu.Unlock()
	now := time.Now()
	maps.DeleteFunc(as.byID, func(_ string, a attempt) bool { return !now.Before(a.expires) })
	as.byID[id] = a
	return id
}

// take removes attempt id, and returns it while it is open for a sign-in
// like for.
func (as *attempts) take(id string, like attempt) (attempt, bool) {
	as.mu.Lock()
	defer as.mu.Unlock()
	a, ok := as.byID[id]
	delete(as.byID, id)
	return a, ok && time.Now().Before(a.expires) && a.node == like.node && a.definition == like.definition && a.username == like.username
}

// signInOf is the sign-in a captcha of idx on node is for.
func signInOf(idx Indexer, node role.NodeID) attempt {
	d, _ := idx.Source.(Definition)
	return attempt{node: node, definition: d.ID, username: d.Plain["username"]}
}

// Captcha fetches a captcha on the Node the Indexer searches from and
// keeps its attempt open for Timings.Captcha. It fails with ErrNoCaptcha,
// *TestFailedError, *WaitError, *UnavailableError or ErrLate, and as a
// Draft's Test does.
func (x *Indexers) Captcha(ctx context.Context, _ identity.Admin, f CaptchaFor) (Captcha, error) {
	if f.Draft != nil {
		idx, session, err := x.drafted(ctx, *f.Draft)
		if err != nil {
			return Captcha{}, err
		}
		node, wait := route(x.cfg.Roster, idx, Status{State: OK})
		if wait != nil {
			return Captcha{}, &WaitError{Outcome: *wait}
		}
		return x.captcha(ctx, idx, node, session)
	}
	idx, err := Get(ctx, x.cfg.Library, f.Indexer)
	if err != nil {
		return Captcha{}, err
	}
	node, err := x.where(ctx, idx)
	if err != nil {
		return Captcha{}, err
	}
	return x.captcha(ctx, idx, node, nil)
}

// captcha fetches a captcha of idx on node, with session in place of its
// stored one unless nil. The job reserves no query row and records no
// status, since it only reads the sign-in page.
func (x *Indexers) captcha(ctx context.Context, idx Indexer, node role.NodeID, session *searching.Session) (Captcha, error) {
	if d, ok := idx.Source.(Definition); !ok || !d.Captcha {
		return Captcha{}, ErrNoCaptcha
	}
	j, err := x.prepare(ctx, QueryID{UUID: storage.NewUUID()}, idx, UseTest, node, searching.CaptchaCall{})
	if err != nil {
		return Captcha{}, err
	}
	j.untracked = true
	if session != nil {
		j.wire.Session = session
	}
	a, err := x.ask(ctx, j)
	switch {
	case err != nil:
		return Captcha{}, err
	case a.Outcome != nil:
		return Captcha{}, x.testFailed(idx, node, a, Status{})
	case a.Captcha == nil:
		return Captcha{}, ErrNoCaptcha
	}
	at := signInOf(idx, node)
	at.preLogin, at.expires = a.Captcha.PreLogin, time.Now().Add(x.cfg.Timings.Captcha)
	return Captcha{Attempt: x.attempts.keep(at), Image: a.Captcha.Image, ContentType: a.Captcha.ContentType, Expires: at.expires}, nil
}

// testCall is the Test of idx on node, signing in with c when the Admin
// answered a captcha. It fails with ErrCaptchaExpired when c's attempt
// isn't open for that sign-in.
func (x *Indexers) testCall(idx Indexer, node role.NodeID, c *CaptchaAnswer) (searching.TestCall, error) {
	if c == nil {
		return searching.TestCall{}, nil
	}
	a, ok := x.attempts.take(c.Attempt, signInOf(idx, node))
	if !ok {
		return searching.TestCall{}, ErrCaptchaExpired
	}
	return searching.TestCall{Captcha: &searching.CaptchaAnswer{PreLogin: a.preLogin, Text: c.Text}}, nil
}

// withCaptcha is err with a new captcha when it is a Test of idx on node
// that failed at a sign-in that shows one, since each captcha is answered
// once.
func (x *Indexers) withCaptcha(ctx context.Context, idx Indexer, node role.NodeID, session *searching.Session, err error) error {
	var failed *TestFailedError
	if d, ok := idx.Source.(Definition); !ok || !d.Captcha || !errors.As(err, &failed) || failed.Outcome.Kind != IsStopped ||
		failed.Outcome.Cause != searching.CaptchaNeeded && failed.Outcome.Cause != searching.CredentialsRejected {
		return err
	}
	c, cerr := x.captcha(ctx, idx, node, session)
	if cerr != nil {
		x.cfg.Log.Warn("Couldn't fetch a new captcha after a failed Test", "indexer", idx.ID.String(), "err", cerr.Error())
		return err
	}
	failed.Captcha = &c
	return err
}
