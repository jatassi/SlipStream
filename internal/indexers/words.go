package indexers

import (
	"cmp"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/works"
)

// stopSaid is what a stop means, as its own sentence: %[1]s is the
// Indexer, %[2]s the site's own words, %[3]s the Node it was met on.
var stopSaid = map[Cause]string{
	searching.CredentialsRejected:  "%[1]s rejected the username or password.",
	searching.CaptchaNeeded:        "%[1]s needs a captcha to sign in again.",
	searching.ChallengeUnsolved:    "%[1]s is behind a browser challenge, and %[3]s has no Challenge solver.",
	searching.CookieExpired:        "%[1]s no longer accepts the browser cookie.",
	searching.DefinitionUnloadable: "%[1]s's definition doesn't load.",
	searching.AccountRefused:       "%[1]s refused the API key with '%[2]s'.",
	searching.RequestRejected:      "%[1]s refused a search with '%[2]s'.",
}

const solverTriedSaid = "%[3]s's Challenge solver couldn't clear %[1]s's challenge."

// stopFormFixes is how the Admin clears a stop from the Indexer's form,
// said after the stop's cause: %[3]s is the Node it was met on.
var stopFormFixes = map[Cause]string{
	searching.CredentialsRejected:  "Check them and test again.",
	searching.CaptchaNeeded:        "Type the code and test again.",
	searching.ChallengeUnsolved:    "Set one on %[3]s, or paste a browser cookie.",
	searching.CookieExpired:        "Paste a new one and test again.",
	searching.DefinitionUnloadable: "Replace the custom definition, or wait for a catalog fix.",
	searching.AccountRefused:       "Check the account, then test again.",
	searching.RequestRejected:      "Test again after the feed is fixed.",
}

// refusalWords is why an RSS feed can't be used, for the form.
var refusalWords = map[searching.RSSRefusal]string{
	searching.RSSNoItems:      "The feed has no items.",
	searching.RSSNoTitle:      "Some items have no title.",
	searching.RSSNoLink:       "Some items have no download link.",
	searching.RSSRepeatedGUID: "Some items repeat the same GUID.",
	searching.RSSNoSize:       "Some items have no size. Turn on Allow items with no size to use the feed anyway.",
}

// refusedSignIn is a sign-in the site refused in its own words, which may
// be about the captcha's code as well as the password.
const (
	refusedSignInSaid    = "%[1]s refused the sign-in with '%[2]s'."
	refusedSignInFormFix = "Check the username, password and any code, then test again."
	refusedSignInItemFix = "Check the username, password and any code on the Indexer, then test it."
)

// refusedSignIn reports whether a stop with cause and the site's own text
// is a sign-in the site refused in its own words.
func refusedSignIn(cause Cause, text string) bool {
	return cause == searching.CredentialsRejected && text != ""
}

// TestFailedError is a Test the site answered with an outcome other than
// ok.
type TestFailedError struct {
	Outcome Outcome
	// Sentence is the cause and its fix, for the form.
	Sentence string
	// Captcha is a new one to answer when the Test failed at a sign-in
	// that shows a captcha.
	Captcha *Captcha
}

func (e *TestFailedError) Error() string { return "the Indexer's Test ended " + string(e.Outcome.Kind) }

// testFailed is the TestFailedError of a Test of idx run on node that the
// site answered with outcome, and the status the answer left.
func (x *Indexers) testFailed(idx Indexer, node role.NodeID, a searching.Answer, st Status) error {
	o := outcomeOf(a, st, idx.Home)
	var sentence string
	var limited *searching.Limited
	var stopped *searching.Stopped
	switch {
	case errors.As(a.Outcome, &stopped):
		said, fix := stopSaid[stopped.Cause], stopFormFixes[stopped.Cause]
		switch {
		case stopped.SolverTried:
			said, fix = solverTriedSaid, "Paste a browser cookie and test again."
		case refusedSignIn(stopped.Cause, stopped.Text):
			said, fix = refusedSignInSaid, refusedSignInFormFix
		}
		sentence = fmt.Sprintf(said+" "+fix, idx.Name, stopped.Text, x.cfg.Roster.Name(node))
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

// stoppedWords is a stop in words, as the Indexer's status and its line in
// a search: %[1]s is the Indexer, %[2]s the site's own words, %[3]s the
// Node it was met on.
var stoppedWords = map[Cause]string{
	searching.CredentialsRejected:  "Stopped because %[1]s rejected the username or password.",
	searching.CaptchaNeeded:        "Stopped because %[1]s needs a captcha to sign in again.",
	searching.ChallengeUnsolved:    "Stopped because %[1]s is behind a browser challenge and %[3]s has no Challenge solver.",
	searching.CookieExpired:        "Stopped because %[1]s no longer accepts the browser cookie.",
	searching.DefinitionUnloadable: "Stopped because %[1]s's definition doesn't load.",
	searching.AccountRefused:       "Stopped because %[1]s refused the API key with '%[2]s'.",
	searching.RequestRejected:      "Stopped because %[1]s refused a search with '%[2]s'.",
}

// stoppedFixes is how the Admin clears a stop from the Indexer's page:
// {node} is the Node it was met on.
var stoppedFixes = map[Cause]string{
	searching.CredentialsRejected:  "Update the username or password with Edit. Saving tests them.",
	searching.CaptchaNeeded:        "Test the Indexer and type the code.",
	searching.ChallengeUnsolved:    "Set a Challenge solver on {node}, or paste a browser cookie with Edit.",
	searching.CookieExpired:        "Paste a new browser cookie with Edit.",
	searching.DefinitionUnloadable: "Replace the custom definition, or wait for a catalog fix.",
	searching.AccountRefused:       "Check the account, then test the Indexer.",
	searching.RequestRejected:      "Test the Indexer after the feed is fixed.",
}

const (
	solverTriedStopped   = "Stopped because %[3]s's Challenge solver couldn't clear %[1]s's challenge."
	solverTriedFix       = "Paste a browser cookie with Edit."
	refusedSignInStopped = "Stopped because %[1]s refused the sign-in with '%[2]s'."
)

// stoppedFormat is the stoppedWords format of a stop with cause and the
// site's own text.
func stoppedFormat(cause Cause, text string) string {
	if refusedSignIn(cause, text) {
		return refusedSignInStopped
	}
	return stoppedWords[cause]
}

// words is v's status as a sentence, and what the Admin can do about it,
// "" when nothing.
func (x *Indexers) words(v View) (sentence, fix string) {
	name := v.Name
	if !v.On {
		return name + " is off.", ""
	}
	if w := v.Waits; w != nil {
		return x.waitWords(v, *w)
	}
	switch {
	case v.Status.State == Failing:
		return fmt.Sprintf("The last request to %s failed. The next search tries it again.", name), ""
	case v.Status.Success.IsZero():
		return fmt.Sprintf("SlipStream hasn't searched %s yet.", name), ""
	}
	return name + " is working.", ""
}

func (x *Indexers) waitWords(v View, w Outcome) (sentence, fix string) {
	name := v.Name
	switch w.Kind {
	case IsStopped:
		node := x.cfg.Roster.Name(w.Node)
		stopped, fix := stoppedFormat(w.Cause, w.Text), stoppedFixes[w.Cause]
		if v.Status.SolverTried {
			stopped, fix = solverTriedStopped, solverTriedFix
		}
		return fmt.Sprintf(stopped, name, w.Text, node), strings.ReplaceAll(fix, "{node}", node)
	case IsLimited:
		return fmt.Sprintf("%s is limiting requests until %s.", name, x.clock(w.Until)), ""
	case IsFailing:
		return fmt.Sprintf("%s isn't responding. SlipStream retries at %s.", name, x.clock(w.Until)), ""
	case LimitReached:
		return fmt.Sprintf("%s reached its search limit. SlipStream searches it again at %s.", name, x.clock(w.Until)), ""
	case Waiting:
		home := x.cfg.Roster.Name(w.Node)
		var fix string
		if w.Lapsed && v.SignsIn && !v.AllowFailover && v.Status.State != HomeOnly {
			fix = "Allow failover to search from another Node."
		}
		if w.Since.IsZero() {
			return fmt.Sprintf("%s is waiting for %s.", name, home), fix
		}
		return fmt.Sprintf("%s is waiting for %s, which has been offline since %s.", name, home, x.clock(w.Since)), fix
	case Found, NoReleases, Late, NoCategories, Removed:
	}
	return name + " is working.", ""
}

// Line is how Indexer name stood in a search of an Item of Content Type
// typ: o in words, or "Searching" while its job runs and o is nil.
func (x *Indexers) Line(name string, typ works.TypeID, o *Outcome) string {
	if o == nil {
		return "Searching"
	}
	switch o.Kind {
	case Found:
		if o.Releases == 1 {
			return "Found 1 Release"
		}
		return fmt.Sprintf("Found %d Releases", o.Releases)
	case NoReleases:
		return "No Releases"
	case IsLimited, LimitReached:
		if o.Until.IsZero() {
			return "Limit reached"
		}
		return "Limit reached until " + x.clock(o.Until)
	case Waiting:
		return "Waiting for " + x.cfg.Roster.Name(o.Node)
	case IsFailing:
		return x.retries(name+" isn't responding.", o.Until)
	case IsStopped:
		return fmt.Sprintf(stoppedFormat(o.Cause, o.Text), name, o.Text, x.cfg.Roster.Name(o.Node))
	case NoCategories:
		d, _ := x.cfg.Works.Describe(typ)
		return "Skipped because it has no " + strings.ToLower(d.Work.One) + " categories"
	case Late:
		return "Didn't finish in time"
	case Removed:
		return "Deleted since this search"
	}
	return string(o.Kind)
}

// fetchWaitWords is why a fetch from Indexer name waits, o being the
// gate's limited, failing or waiting Outcome.
func (x *Indexers) fetchWaitWords(name string, o Outcome) string {
	if o.Kind == Waiting {
		return fmt.Sprintf("Waiting for %s to fetch the .torrent from %s.", x.cfg.Roster.Name(o.Node), name)
	}
	return x.retries("Waiting to fetch the .torrent from "+name+".", o.Until)
}

// retries is sentence followed by when SlipStream retries, at until.
func (x *Indexers) retries(sentence string, until time.Time) string {
	if until.IsZero() {
		return sentence + " SlipStream retries automatically."
	}
	return sentence + " SlipStream retries at " + x.clock(until) + "."
}

// stopItemFixes is how the Admin clears a stop, said in its Health item
// after the stop's cause: %[3]s is the Node it was met on.
var stopItemFixes = map[Cause]string{
	searching.CredentialsRejected:  "Update them on the Indexer and test it.",
	searching.CaptchaNeeded:        "Open the Indexer and type the code.",
	searching.ChallengeUnsolved:    "Set one on %[3]s, or paste a browser cookie on the Indexer.",
	searching.CookieExpired:        "Paste a new one on the Indexer.",
	searching.DefinitionUnloadable: "Replace the custom definition, or wait for a catalog fix.",
	searching.AccountRefused:       "Check the account, then test the Indexer.",
	searching.RequestRejected:      "Test the Indexer after the feed is fixed.",
}

// stopItemWords is the Health item of Indexer name stopped at st.
func (x *Indexers) stopItemWords(name string, st Status) string {
	said, fix := stopSaid[st.Cause], stopItemFixes[st.Cause]
	switch {
	case st.SolverTried:
		said, fix = solverTriedSaid, "Paste a browser cookie on the Indexer."
	case refusedSignIn(st.Cause, st.Text):
		said, fix = refusedSignInSaid, refusedSignInItemFix
	}
	return fmt.Sprintf(said+" "+fix, name, st.Text, x.cfg.Roster.Name(st.Node))
}

// failingItemWords is the Health item of Indexer name failing at st
// longer than the long backoff.
func (x *Indexers) failingItemWords(name string, st Status, now time.Time) string {
	until := st.Until
	if !now.Before(until) {
		until = time.Time{}
	}
	return x.retries(fmt.Sprintf("Every request to %s has failed since %s.", name, x.clock(st.Since)), until)
}

// unservedWords is the Health item of a Content Type whose Works, many,
// no usable Indexer searches for.
func unservedWords(many string) string {
	return "No Indexer can search for " + strings.ToLower(many) + ". Add an Indexer or turn one on."
}
