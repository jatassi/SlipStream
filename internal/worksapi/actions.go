package worksapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/targetsapi"
)

type monitoredInput struct {
	Body monitoredChange
}

type monitoredChange struct {
	WorkIDs []string `json:"workIds" format:"uuid" minItems:"1" maxItems:"20000" nullable:"false" doc:"A repeated id counts once"`
	On      bool     `json:"on" doc:"Monitor every Item of each Work, or none"`
}

type profileInput struct {
	targetsapi.Changing
	Body profileChange
}

type profileChange struct {
	WorkIDs   []string `json:"workIds" format:"uuid" minItems:"1" maxItems:"20000" nullable:"false" doc:"A repeated id counts once"`
	ProfileID string   `json:"profileId" format:"uuid" doc:"The Profile each Work wants from; a Work with its own Edition list keeps it"`
}

type profileChanged struct {
	Impact   targetsapi.Impact    `json:"impact"`
	Outcomes []api.OneOf[outcome] `json:"outcomes" nullable:"false" doc:"One per Work, in request order"`
}

type profileChangedOutput struct{ Body profileChanged }

type rootInput struct {
	Body rootChange
}

type rootChange struct {
	WorkIDs []string `json:"workIds" format:"uuid" minItems:"1" maxItems:"20000" nullable:"false" doc:"A repeated id counts once"`
	RootID  string   `json:"rootId" format:"uuid" doc:"The root each Work moves to; a Work with files stays where it is"`
}

type refreshInput struct {
	Body refreshing
}

type refreshing struct {
	WorkIDs []string `json:"workIds" format:"uuid" minItems:"1" maxItems:"20000" nullable:"false" doc:"A repeated id counts once"`
}

type outcomeList struct {
	Outcomes []api.OneOf[outcome] `json:"outcomes" nullable:"false" doc:"One per Work, in request order"`
}

type outcomeListOutput struct{ Body outcomeList }

func (h handlers) setWorksMonitored(ctx context.Context, by identity.Admin, in *monitoredInput) (*outcomeListOutput, error) {
	ws, err := workIDsOf(in.Body.WorkIDs)
	if err != nil {
		return nil, err
	}
	if _, err := composer(h.shelf).Monitor(ctx, by, ws, in.Body.On); err != nil {
		return nil, answer(err)
	}
	return nil, notImplemented
}

func (h handlers) setWorksProfile(ctx context.Context, by identity.Admin, in *profileInput) (*profileChangedOutput, error) {
	ws, err := workIDsOf(in.Body.WorkIDs)
	if err != nil {
		return nil, err
	}
	u, err := storage.ParseUUID(in.Body.ProfileID)
	if err != nil {
		return nil, profileNotFound
	}
	if _, _, err := composer(h.shelf).SetProfile(ctx, by, ws, targets.ProfileID{UUID: u}, in.Mode()); err != nil {
		return nil, answer(err)
	}
	return nil, notImplemented
}

func (h handlers) setWorksRoot(ctx context.Context, by identity.Admin, in *rootInput) (*outcomeListOutput, error) {
	ws, err := workIDsOf(in.Body.WorkIDs)
	if err != nil {
		return nil, err
	}
	u, err := storage.ParseUUID(in.Body.RootID)
	if err != nil {
		return nil, rootNotFound
	}
	if _, err := composer(h.shelf).SetRoot(ctx, by, ws, roots.ID{UUID: u}); err != nil {
		return nil, answer(err)
	}
	return nil, notImplemented
}

func (h handlers) refreshWorks(ctx context.Context, by identity.Admin, in *refreshInput) (*outcomeListOutput, error) {
	ws, err := workIDsOf(in.Body.WorkIDs)
	if err != nil {
		return nil, err
	}
	if _, err := composer(h.shelf).Refresh(ctx, by, ws); err != nil {
		return nil, answer(err)
	}
	return nil, notImplemented
}
