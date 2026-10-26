package worksapi

import (
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/works"
)

type outcome struct{}

func (outcome) Variants() (discriminator string, variants []any) {
	return "outcome", []any{doneOutcome{}, skippedOutcome{}, failedOutcome{}}
}

type doneOutcome struct {
	WorkID  string `json:"workId" format:"uuid"`
	Outcome string `json:"outcome" enum:"done"`
}

type skippedOutcome struct {
	WorkID  string `json:"workId" format:"uuid"`
	Outcome string `json:"outcome" enum:"skipped"`
	Reason  string `json:"reason" enum:"already,work_has_files,work_gone_upstream,work_not_found" doc:"Why the action left the Work as it was"`
}

type failedOutcome struct {
	WorkID  string         `json:"workId" format:"uuid"`
	Outcome string         `json:"outcome" enum:"failed"`
	Problem outcomeProblem `json:"problem"`
}

type outcomeProblem struct {
	Code   string `json:"code" doc:"The code a request for this Work alone would have been refused with"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

var storageUnreachable = api.NewCode(http.StatusConflict, "storage_unreachable", "The Work's files can't be deleted")

var skipReasons = []struct {
	why    error
	reason string
}{
	{library.ErrAlready, "already"},
	{roots.ErrHasFiles, "work_has_files"},
	{works.ErrGoneUpstream, "work_gone_upstream"},
	{works.ErrNoWork, "work_not_found"},
}

// outcomesOf answers each outcome on the wire. A failure no code answers
// is an internal fault, not the Work's outcome, so it fails the whole
// request and the router logs it.
func outcomesOf(os []library.Outcome) ([]api.OneOf[outcome], error) {
	out := make([]api.OneOf[outcome], len(os))
	for i, o := range os {
		id := o.Work.String()
		switch o.Result {
		case library.Done:
			out[i] = api.NewOneOf[outcome](doneOutcome{WorkID: id, Outcome: "done"})
		case library.Skipped:
			out[i] = api.NewOneOf[outcome](skippedOutcome{WorkID: id, Outcome: "skipped", Reason: reasonOf(o.Why)})
		case library.Failed:
			p, err := problemOf(o.Why)
			if err != nil {
				return nil, err
			}
			out[i] = api.NewOneOf[outcome](failedOutcome{WorkID: id, Outcome: "failed", Problem: p})
		}
	}
	return out, nil
}

func reasonOf(why error) string {
	for _, s := range skipReasons {
		if errors.Is(why, s.why) {
			return s.reason
		}
	}
	panic("worksapi: a Work was skipped for " + why.Error() + ", which is no skip reason")
}

// problemOf is the problem a request for the Work alone would have been
// refused with, the error itself when no code answers it.
func problemOf(why error) (outcomeProblem, error) {
	err := answer(why)
	if away, ok := errors.AsType[*roots.UnreachableError](why); ok {
		err = storageUnreachable.Because(unreachableWords(away))
	}
	p, ok := api.ProblemOf(err)
	if !ok {
		return outcomeProblem{}, why
	}
	return outcomeProblem{Code: p.Code, Title: p.Title, Detail: p.Detail}, nil
}

func unreachableWords(e *roots.UnreachableError) string {
	if e.State == roots.WrittenOff {
		return e.Node + "'s Storage was written off, so this Work's files there can't be deleted. Delete the Work and leave its files."
	}
	return e.Node + " is offline, so this Work's files there can't be deleted. Delete the Work and leave its files, or try again once " + e.Node + " is back."
}
