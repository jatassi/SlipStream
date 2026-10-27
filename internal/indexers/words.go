package indexers

import (
	"cmp"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
)

// stopWords is what a stop means and how the Admin clears it from the
// Indexer's form: %[1]s is the Indexer, %[2]s the site's own words, %[3]s
// the Node it was met on.
var stopWords = map[Cause]string{
	searching.CredentialsRejected:  "%[1]s rejected the username or password. Check them and test again.",
	searching.CaptchaNeeded:        "%[1]s needs a captcha to sign in again. Type the code and test again.",
	searching.ChallengeUnsolved:    "%[1]s is behind a browser challenge, and %[3]s has no Challenge solver. Set one on %[3]s, or paste a browser cookie.",
	searching.CookieExpired:        "%[1]s no longer accepts the browser cookie. Paste a new one and test again.",
	searching.DefinitionUnloadable: "%[1]s's definition doesn't load. Replace the custom definition, or wait for a catalog fix.",
	searching.AccountRefused:       "%[1]s refused the API key with '%[2]s'. Check the account, then test again.",
	searching.RequestRejected:      "%[1]s refused a search with '%[2]s'. Test again after the feed is fixed.",
}

const solverTriedWords = "%[3]s's Challenge solver couldn't clear %[1]s's challenge. Paste a browser cookie and test again."

// TestFailedError is a Test the site answered with an outcome other than
// ok.
type TestFailedError struct {
	Outcome Outcome
	// Sentence is the cause and its fix, for the form.
	Sentence string
}

func (e *TestFailedError) Error() string { return "the Indexer's Test ended " + string(e.Outcome.Kind) }

// testFailed is the TestFailedError of a Test of idx run on node that the
// site answered with outcome, and the status the answer left.
func (x *Indexers) testFailed(idx Indexer, node role.NodeID, a searching.Answer, st Status) error {
	o := outcomeOf(a, st)
	var sentence string
	var limited *searching.Limited
	var stopped *searching.Stopped
	switch {
	case errors.As(a.Outcome, &stopped):
		words := stopWords[stopped.Cause]
		if stopped.SolverTried {
			words = solverTriedWords
		}
		sentence = fmt.Sprintf(words, idx.Name, stopped.Text, x.cfg.Roster.Name(node))
	case errors.As(a.Outcome, &limited):
		until := o.Until
		if until.IsZero() {
			until = time.Now().Add(cmp.Or(limited.RetryAfter, x.cfg.Timings.Limited))
		}
		sentence = fmt.Sprintf("%s is limiting requests. Test again after %s.", idx.Name, x.clock(until))
	default:
		sentence = fmt.Sprintf("%s failed the Test with '%s'. Check the address and test again.", idx.Name, a.Outcome.Error())
	}
	return &TestFailedError{Outcome: o, Sentence: sentence}
}

// Unreached is why a call didn't reach the site, as a sentence: a Node
// that couldn't take it, or a job that ran past its deadline.
func (x *Indexers) Unreached(err error) string {
	var wait *WaitError
	var unavailable *UnavailableError
	switch {
	case errors.As(err, &wait):
		name := x.cfg.Roster.Name(wait.Outcome.Node)
		if wait.Outcome.Since.IsZero() {
			return name + " can't search now. Test again once it holds Searching."
		}
		return fmt.Sprintf("%s has been offline since %s. Test again once it's back.", name, x.clock(wait.Outcome.Since))
	case errors.Is(err, ErrLate):
		return fmt.Sprintf("The Test didn't finish within %d seconds. Test again.", int(x.cfg.Timings.Job.Seconds()))
	case errors.As(err, &unavailable):
		return "SlipStream couldn't reach the Searching Node. Test again once it's back."
	}
	return err.Error()
}

func (x *Indexers) clock(t time.Time) string { return x.inZone(t).Format("15:04") }

// day is t's date in sentences, such as "3 Oct".
func (x *Indexers) day(t time.Time) string { return x.inZone(t).Format("2 Jan") }

func (x *Indexers) inZone(t time.Time) time.Time {
	zone := x.cfg.Zone
	if zone == nil {
		zone = time.Local
	}
	return t.In(zone)
}
