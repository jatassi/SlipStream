package targetsapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type entry struct {
	TargetID string `json:"targetId" format:"uuid"`
	Sourcing string `json:"sourcing" enum:"download" doc:"How the Target gets its file"`
}

// editions is an Edition list. Any cut is the only kind until cuts can
// be chosen.
type editions struct {
	Kind string `json:"kind" enum:"any" doc:"Any cut: one file per Target, of whatever cut"`
}

var anyCut = editions{Kind: "any"}

// Profile is exported so the document carries the fields a Profile's
// detail embeds.
type Profile struct {
	ID          string   `json:"id" format:"uuid"`
	Name        string   `json:"name"`
	ContentType string   `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Default     bool     `json:"default" doc:"Whether a Work added with no Profile takes it"`
	Targets     []entry  `json:"targets" nullable:"false" doc:"In Profile order: a file is named after the first it fills"`
	Editions    editions `json:"editions"`
	Works       int      `json:"works" minimum:"0" doc:"The Works wanting from it"`
}

func profileOf(p targets.Profile) Profile {
	out := Profile{ID: p.ID.String(), Name: p.Name, ContentType: string(p.ContentType), Default: p.Default, Targets: []entry{}, Editions: anyCut, Works: p.Works}
	for _, e := range p.Entries {
		out.Targets = append(out.Targets, entry{TargetID: e.Target.String(), Sourcing: string(e.Sourcing)})
	}
	return out
}

type estimate struct {
	TargetID     string   `json:"targetId" format:"uuid"`
	MiBPerMinute *float64 `json:"mibPerMinute" minimum:"0" doc:"What the files filling the Target average, null with no estimate yet"`
	Bytes        *int64   `json:"bytes" minimum:"0" doc:"That rate over the runtime of the Profile's Works, null with no estimate yet"`
}

type profileDetail struct {
	Profile
	Estimates []estimate `json:"estimates" nullable:"false" doc:"One per Target, in Profile order. A Target no file of known runtime fills yet has no estimate"`
}

type newProfile struct {
	Name        string   `json:"name" minLength:"1" maxLength:"100"`
	ContentType string   `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Targets     []entry  `json:"targets" nullable:"false" uniqueItems:"true" doc:"In Profile order; none is a Profile whose Works want nothing"`
	Editions    editions `json:"editions"`
}

type createProfileInput struct{ Body newProfile }

type profileInput struct {
	ProfileID string `path:"profileId" format:"uuid"`
}

type profileOutput struct{ Body Profile }

type profileDetailOutput struct{ Body profileDetail }

type profileList struct {
	Profiles []Profile `json:"profiles" nullable:"false" doc:"By Content Type, then name"`
}

type profileListOutput struct{ Body profileList }

type workInput struct {
	WorkID string `path:"workId" format:"uuid"`
}

type wants struct {
	ProfileID   string   `json:"profileId" format:"uuid"`
	ProfileName string   `json:"profileName"`
	Editions    editions `json:"editions" doc:"The effective Edition list"`
	From        string   `json:"from" enum:"profile,work" doc:"Whether the Edition list is the Profile's or the Work's own"`
}

type wantsOutput struct{ Body wants }

func (h handlers) createProfile(ctx context.Context, by identity.Admin, in *createProfileInput) (*profileOutput, error) {
	s := targets.ProfileSpec{Name: in.Body.Name, ContentType: works.TypeID(in.Body.ContentType)}
	for _, e := range in.Body.Targets {
		u, err := storage.ParseUUID(e.TargetID)
		if err != nil {
			return nil, targetNotFound
		}
		s.Entries = append(s.Entries, targets.Entry{Target: targets.TargetID{UUID: u}, Sourcing: targets.Sourcing(e.Sourcing)})
	}
	p, err := open(h.goals).CreateProfile(ctx, by, s)
	if err != nil {
		return nil, answer(err)
	}
	return &profileOutput{Body: profileOf(p)}, nil
}

func (h handlers) listProfiles(ctx context.Context, by identity.Admin, _ *struct{}) (*profileListOutput, error) {
	list, err := open(h.goals).ListProfiles(ctx, by)
	if err != nil {
		return nil, err
	}
	out := profileList{Profiles: []Profile{}}
	for _, p := range list {
		out.Profiles = append(out.Profiles, profileOf(p))
	}
	return &profileListOutput{Body: out}, nil
}

func profileID(s string) (targets.ProfileID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return targets.ProfileID{}, noSuchProfile
	}
	return targets.ProfileID{UUID: u}, nil
}

func (h handlers) getProfile(ctx context.Context, by identity.Admin, in *profileInput) (*profileDetailOutput, error) {
	id, err := profileID(in.ProfileID)
	if err != nil {
		return nil, err
	}
	p, err := open(h.goals).GetProfile(ctx, by, id)
	if err != nil {
		return nil, answer(err)
	}
	estimates, err := shelf(h.goals).Estimates(ctx, by, p)
	if err != nil {
		return nil, err
	}
	out := profileDetail{Profile: profileOf(p), Estimates: []estimate{}}
	for _, e := range p.Entries {
		row := estimate{TargetID: e.Target.String()}
		if est, ok := estimates[e.Target]; ok {
			row.MiBPerMinute, row.Bytes = &est.MiBPerMinute, &est.Bytes
		}
		out.Estimates = append(out.Estimates, row)
	}
	return &profileDetailOutput{Body: out}, nil
}

func (h handlers) makeDefaultProfile(ctx context.Context, by identity.Admin, in *profileInput) (*profileOutput, error) {
	id, err := profileID(in.ProfileID)
	if err != nil {
		return nil, err
	}
	p, err := open(h.goals).MakeDefault(ctx, by, id)
	if err != nil {
		return nil, answer(err)
	}
	return &profileOutput{Body: profileOf(p)}, nil
}

func (h handlers) getWorkWants(ctx context.Context, by identity.Admin, in *workInput) (*wantsOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	g, err := open(h.goals).Goals(ctx, by, works.WorkID{UUID: u})
	if err != nil {
		return nil, answer(err)
	}
	return &wantsOutput{Body: wants{ProfileID: g.Profile.ID.String(), ProfileName: g.Profile.Name, Editions: anyCut, From: "profile"}}, nil
}
