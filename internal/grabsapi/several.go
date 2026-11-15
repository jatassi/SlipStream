package grabsapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var tooManyWorks = api.NewCode(http.StatusUnprocessableEntity, "too_many_works", "Search at most 5,000 Works at once")

type severalInput struct{ Body severalBody }

type severalBody struct {
	WorkIDs []string      `json:"workIds,omitempty" format:"uuid" doc:"The Works to search; send this or wanted"`
	Wanted  *wantedFilter `json:"wanted,omitempty" doc:"Every Work Wanted lists for this filter; send this or workIds"`
}

type wantedFilter struct {
	ContentType string   `json:"contentType,omitempty" pattern:"^[a-z][a-z0-9]{1,15}$" doc:"Left out, every Content Type"`
	Reasons     []string `json:"reasons,omitempty" enum:"failed,missing,upgrade,gone" doc:"Left out, every reason but gone"`
}

type severalQueued struct {
	Queued   int           `json:"queued" minimum:"0" doc:"Works with at least one group newly queued"`
	Merged   int           `json:"merged" minimum:"0" doc:"Works whose every group was already queued or searching"`
	LeftGone int           `json:"leftGone" minimum:"0" doc:"Works left out because only gone files want a Release"`
	Works    []workOutcome `json:"works" nullable:"false" doc:"One per Work, in request order"`
}

type workOutcome struct {
	WorkID  string  `json:"workId" format:"uuid"`
	Outcome string  `json:"outcome" enum:"queued,merged,skipped"`
	Reason  *string `json:"reason" enum:"nothing_wanted,gone_only,no_root,not_searchable" doc:"Why it was skipped; null unless skipped"`
}

type severalQueuedOutput struct{ Body severalQueued }

var searchSeveralWorksOp = api.Op{
	ID:      "searchSeveralWorks",
	Method:  http.MethodPost,
	Path:    "/works/search",
	Summary: "Queue a search of several Works, by id or by a Wanted filter, behind every search already queued",
	Status:  http.StatusAccepted,
	Errors:  []*api.Code{reconciling, tooManyWorks},
}

const mostWorks = 5000

var wantedReasons = map[string]files.Reason{
	"failed": files.ReasonFailed, "missing": files.ReasonMissing, "upgrade": files.ReasonUpgrade, "gone": files.ReasonGone,
}

func (h handlers) searchSeveralWorks(ctx context.Context, by identity.Admin, in *severalInput) (*severalQueuedOutput, error) {
	s, err := severalOf(in.Body)
	if err != nil {
		return nil, err
	}
	q, err := open(h.p).SearchSeveral(ctx, by, s)
	if errors.Is(err, grabs.ErrReconciling) {
		return nil, reconciling.Because("Search again once Reconciling finishes.")
	}
	if err != nil {
		return nil, err
	}
	out := severalQueued{Queued: q.Queued, Merged: q.Merged, LeftGone: q.LeftGone, Works: make([]workOutcome, len(q.Works))}
	for i, o := range q.Works {
		w := workOutcome{WorkID: o.Work.String(), Outcome: "queued"}
		switch {
		case o.Skipped != "":
			w.Outcome, w.Reason = "skipped", new(string(o.Skipped))
		case o.Merged:
			w.Outcome = "merged"
		}
		out.Works[i] = w
	}
	return &severalQueuedOutput{Body: out}, nil
}

// severalOf is the search b asks for: its Works by id, or Wanted's for
// its filter.
func severalOf(b severalBody) (grabs.Several, error) {
	switch {
	case (b.WorkIDs == nil) == (b.Wanted == nil):
		return grabs.Several{}, api.Invalid.On("workIds", "Send either workIds or wanted.")
	case len(b.WorkIDs) > mostWorks:
		return grabs.Several{}, tooManyWorks
	case b.Wanted != nil:
		f := grabs.WantedFilter{Type: works.TypeID(b.Wanted.ContentType)}
		for _, r := range b.Wanted.Reasons {
			f.Reasons = append(f.Reasons, wantedReasons[r])
		}
		return grabs.Several{Wanted: &f}, nil
	}
	ids := make([]works.WorkID, len(b.WorkIDs))
	for i, s := range b.WorkIDs {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return grabs.Several{}, api.Invalid.On(fmt.Sprintf("workIds[%d]", i), "Use a Work's id.")
		}
		ids[i] = works.WorkID{UUID: u}
	}
	return grabs.Several{Works: ids}, nil
}
