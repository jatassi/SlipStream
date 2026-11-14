package grabsapi

import (
	"context"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
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

func severalRoutes(o *api.Owner, h handlers) {
	api.Register(o, api.Admin, api.Op{
		ID:      "searchSeveralWorks",
		Method:  http.MethodPost,
		Path:    "/works/search",
		Summary: "Queue a search of several Works, by id or by a Wanted filter, behind every search already queued",
		Status:  http.StatusAccepted,
		Errors:  []*api.Code{reconciling, tooManyWorks, notYet},
	}, h.searchSeveralWorks)
}

func (handlers) searchSeveralWorks(context.Context, identity.Admin, *severalInput) (*severalQueuedOutput, error) {
	return nil, notYet
}
