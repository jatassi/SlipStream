package targetsapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	impactChanged    = api.NewCode(http.StatusConflict, "impact_changed", "The change now does more than its preview showed")
	profileIsDefault = api.NewCode(http.StatusConflict, "profile_is_default", "The default Profile can't be deleted")
	profileWrongType = api.NewCode(http.StatusUnprocessableEntity, "profile_wrong_type", "The Profile is for another Content Type")
	profileNotFound  = api.NewCode(http.StatusUnprocessableEntity, "profile_not_found", "No Profile has that id")
)

type unwantedFiles struct {
	Files int   `json:"files" minimum:"0"`
	Bytes int64 `json:"bytes" minimum:"0"`
}

type impact struct {
	Works       int           `json:"works" minimum:"0" doc:"Works whose cells, fills or file names change"`
	OwnList     int           `json:"ownList" minimum:"0" doc:"Works that keep their own Edition list through a change to their Profile's"`
	Unwanted    unwantedFiles `json:"unwanted" doc:"Files that fill a cell now and none after: deleted, or kept as Kept files, as the save chooses"`
	Renamed     int           `json:"renamed" minimum:"0" doc:"Files that still fill a cell and change name"`
	NewlyWanted int           `json:"newlyWanted" minimum:"0" doc:"Cells that become wanted"`
	Downloading int           `json:"downloading" minimum:"0" doc:"Downloads in flight for cells that go; their files follow the save's choice"`
}

func impactOf(i files.Impact) impact {
	return impact{
		Works: i.Works, OwnList: i.OwnList, Unwanted: unwantedFiles{Files: i.Unwanted, Bytes: i.Bytes},
		Renamed: i.Renamed, NewlyWanted: i.NewlyWanted, Downloading: i.Downloading,
	}
}

// Changing is the query every change that can leave files unwanted takes.
// It is exported so the document carries the parameters it embeds.
type Changing struct {
	Preview  bool                `query:"preview" doc:"Answer the change's Impact and save nothing"`
	Unwanted string              `query:"unwanted" enum:"delete,keep" default:"delete" doc:"What becomes of the files the change leaves filling nothing, and of downloads in flight for cells it removes"`
	Confirm  api.Optional[int64] `query:"confirm" minimum:"0" doc:"The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed"`
}

func (c Changing) mode() files.Mode {
	m := files.Mode{Preview: c.Preview, Unwanted: files.Unwanted(c.Unwanted)}
	if c.Confirm.Set {
		m.Confirmed = new(int(c.Confirm.Value))
	}
	return m
}

// changed answers err from a Change.
func changed(err error) error {
	if e, ok := errors.AsType[*files.ImpactChangedError](err); ok {
		return impactChanged.Because(e.Error())
	}
	return answer(err)
}

type updateTargetInput struct {
	TargetID string `path:"targetId" format:"uuid"`
	Changing
	Body newTarget
}

type targetChange struct {
	Impact impact `json:"impact"`
	Target target `json:"target"`
}

type targetChangeOutput struct{ Body targetChange }

func (h handlers) updateTarget(ctx context.Context, by identity.Admin, in *updateTargetInput) (*targetChangeOutput, error) {
	id, err := targetID(in.TargetID)
	if err != nil {
		return nil, err
	}
	s, err := specOf(in.Body)
	if err != nil {
		return nil, err
	}
	i, err := shelf(h.goals).Change(ctx, by, targets.UpdateTarget{ID: id, Spec: s}, in.mode())
	switch {
	case errors.Is(err, targets.ErrNoTarget):
		return nil, noSuchTarget
	case err != nil:
		return nil, changed(err)
	}
	t, err := open(h.goals).GetTarget(ctx, by, id)
	if err != nil {
		return nil, err
	}
	return &targetChangeOutput{Body: targetChange{Impact: impactOf(i), Target: targetOf(t)}}, nil
}

type updateProfileInput struct {
	ProfileID string `path:"profileId" format:"uuid"`
	Changing
	Body newProfile
}

type profileChange struct {
	Impact  impact  `json:"impact"`
	Profile Profile `json:"profile"`
}

type profileChangeOutput struct{ Body profileChange }

func (h handlers) updateProfile(ctx context.Context, by identity.Admin, in *updateProfileInput) (*profileChangeOutput, error) {
	id, err := profileID(in.ProfileID)
	if err != nil {
		return nil, err
	}
	s, err := profileSpecOf(in.Body)
	if err != nil {
		return nil, err
	}
	i, err := shelf(h.goals).Change(ctx, by, targets.UpdateProfile{ID: id, Spec: s}, in.mode())
	if err != nil {
		return nil, wrongType(changed(err))
	}
	return h.profileChanged(ctx, by, i, id)
}

type deleteProfileInput struct {
	ProfileID string `path:"profileId" format:"uuid"`
	MoveTo    string `query:"moveTo" format:"uuid" required:"true" doc:"The Profile of the same Content Type that its Works move to"`
	Changing
}

func (h handlers) deleteProfile(ctx context.Context, by identity.Admin, in *deleteProfileInput) (*profileChangeOutput, error) {
	id, err := profileID(in.ProfileID)
	if err != nil {
		return nil, err
	}
	u, err := storage.ParseUUID(in.MoveTo)
	if err != nil {
		return nil, profileNotFound
	}
	to := targets.ProfileID{UUID: u}
	i, err := shelf(h.goals).Change(ctx, by, targets.DeleteProfile{ID: id, To: to}, in.mode())
	switch {
	case errors.Is(err, targets.ErrNoReplacement):
		return nil, profileNotFound
	case errors.Is(err, targets.ErrProfileIsDefault):
		return nil, profileIsDefault
	case errors.Is(err, targets.ErrMovesToItself):
		return nil, api.Invalid.Fields(api.FieldError{Location: "query.moveTo", Message: err.Error()})
	case err != nil:
		return nil, wrongType(changed(err))
	}
	return h.profileChanged(ctx, by, i, to)
}

func (h handlers) profileChanged(ctx context.Context, by identity.Admin, i files.Impact, id targets.ProfileID) (*profileChangeOutput, error) {
	p, err := open(h.goals).GetProfile(ctx, by, id)
	if err != nil {
		return nil, err
	}
	return &profileChangeOutput{Body: profileChange{Impact: impactOf(i), Profile: profileOf(p)}}, nil
}

// wrongType answers a Profile of another Content Type, which the body or
// query names.
func wrongType(err error) error {
	if errors.Is(err, targets.ErrProfileWrongType) {
		return profileWrongType
	}
	return err
}

type newWants struct {
	ProfileID string              `json:"profileId" format:"uuid"`
	Editions  api.Patch[editions] `json:"editions" doc:"The Work's own Edition list, or null for its Profile's"`
}

type setWantsInput struct {
	WorkID string `path:"workId" format:"uuid"`
	Changing
	Body newWants
}

type wantsChange struct {
	Impact impact `json:"impact"`
	Wants  wants  `json:"wants"`
}

type wantsChangeOutput struct{ Body wantsChange }

func (h handlers) setWorkWants(ctx context.Context, by identity.Admin, in *setWantsInput) (*wantsChangeOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	work := works.WorkID{UUID: u}
	p, err := storage.ParseUUID(in.Body.ProfileID)
	if err != nil {
		return nil, profileNotFound
	}
	c := targets.SetWants{Work: work, Profile: targets.ProfileID{UUID: p}}
	if own := in.Body.Editions.Value(); own != nil {
		list, err := own.parse()
		if err != nil {
			return nil, err
		}
		c.Own = &list
	}
	i, err := shelf(h.goals).Change(ctx, by, c, in.mode())
	switch {
	case errors.Is(err, targets.ErrNoProfile):
		return nil, profileNotFound
	case err != nil:
		return nil, wrongType(changed(err))
	}
	g, err := open(h.goals).Goals(ctx, by, work)
	if err != nil {
		return nil, err
	}
	return &wantsChangeOutput{Body: wantsChange{Impact: impactOf(i), Wants: wantsOf(g)}}, nil
}
