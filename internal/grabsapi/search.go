package grabsapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type searchWorkInput struct {
	WorkID string `path:"workId" format:"uuid"`
	Body   *searchScope
}

type searchScope struct {
	LevelIDs []string `json:"levelIds" format:"uuid" maxItems:"20000" nullable:"false" doc:"Each level stands for every Item under it"`
	ItemIDs  []string `json:"itemIds" format:"uuid" maxItems:"20000" nullable:"false"`
}

type searchQueued struct {
	Groups []queuedGroup `json:"groups" nullable:"false" doc:"Each search group the press queued, in tree order"`
}

type queuedGroup struct {
	Key   string `json:"key" doc:"The search group's key, to compare and never to parse"`
	Label string `json:"label" doc:"The group in words, such as season 3"`
	State string `json:"state" enum:"queued,merged" doc:"merged is a group already queued or searching with this scope"`
}

type searchQueuedOutput struct{ Body searchQueued }

var searchWorkOp = api.Op{
	ID:      "searchWork",
	Method:  http.MethodPost,
	Path:    "/works/{workId}/search",
	Summary: "Queue a search of each wanted Item of a Work, or of some of its levels and Items, to grab the best release",
	Status:  http.StatusAccepted,
	Errors:  []*api.Code{workNotFound, nothingWanted, noIndexer, noRoot, rootUnreachable, notSearchable, reconciling},
}

func (h handlers) searchWork(ctx context.Context, by identity.Admin, in *searchWorkInput) (*searchQueuedOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	scope, err := scopeOf(in.Body)
	if err != nil {
		return nil, err
	}
	groups, err := open(h.p).SearchWork(ctx, by, works.WorkID{UUID: u}, scope)
	if e, ok := errors.AsType[*works.NotInWorkError](err); ok {
		return nil, notInWork(e)
	}
	if errors.Is(err, grabs.ErrReconciling) {
		return nil, reconciling.Because("Search again once Reconciling finishes.")
	}
	if err != nil {
		return nil, answer(err)
	}
	out := searchQueued{Groups: make([]queuedGroup, len(groups))}
	for i, g := range groups {
		out.Groups[i] = queuedGroup{Key: string(g.Key), Label: g.Label, State: "queued"}
		if g.Merged {
			out.Groups[i].State = "merged"
		}
	}
	return &searchQueuedOutput{Body: out}, nil
}

// scopeOf is the search scope body names, the whole Work when it is left
// out.
func scopeOf(body *searchScope) (grabs.Scope, error) {
	if body == nil {
		return grabs.Scope{}, nil
	}
	if len(body.LevelIDs) == 0 && len(body.ItemIDs) == 0 {
		return grabs.Scope{}, api.Invalid.On("levelIds", "Send at least one level or Item id, or leave the body out to search the whole Work.")
	}
	part, err := withinOf(body.LevelIDs, body.ItemIDs)
	if err != nil {
		return grabs.Scope{}, err
	}
	return grabs.Within(part), nil
}

func withinOf(levels, items []string) (works.Within, error) {
	var part works.Within
	for _, s := range levels {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return works.Within{}, api.Invalid
		}
		part.Levels = append(part.Levels, works.LevelID{UUID: u})
	}
	for _, s := range items {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return works.Within{}, api.Invalid
		}
		part.Items = append(part.Items, works.ItemID{UUID: u})
	}
	return part, nil
}

// notInWork is e as the field it names.
func notInWork(e *works.NotInWorkError) error {
	field, noun := "itemIds", "Item"
	if e.Level {
		field, noun = "levelIds", "level"
	}
	return api.Invalid.On(fmt.Sprintf("%s[%d]", field, e.Index), "The Work has no "+noun+" with that id.")
}
