package targetsapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type entry struct {
	TargetID string `json:"targetId" format:"uuid"`
	Sourcing string `json:"sourcing" enum:"download" doc:"How the Target gets its file"`
}

// editions is an Edition list: any cut, or named cuts, which the union
// keeps from mixing.
type editions struct{}

func (editions) Variants() (discriminator string, variants []any) {
	return "kind", []any{anyCut{}, namedCuts{}}
}

type anyCut struct {
	Kind string `json:"kind" enum:"any" doc:"Any cut: one file per Target, of whatever cut"`
}

type namedCuts struct {
	Kind string   `json:"kind" enum:"cuts" doc:"Named cuts: with Standard, each other cut once seen for an Item; without it, every cut outright"`
	Cuts []string `json:"cuts" nullable:"false" minItems:"1" uniqueItems:"true" doc:"Cut keys in list order"`
}

func editionsOf(e targets.Editions) api.OneOf[editions] {
	if e.Any() {
		return api.NewOneOf[editions](anyCut{Kind: "any"})
	}
	out := namedCuts{Kind: "cuts", Cuts: []string{}}
	for _, c := range e.Cuts() {
		out.Cuts = append(out.Cuts, string(c))
	}
	return api.NewOneOf[editions](out)
}

// editionsFrom reads a body's Edition list.
func editionsFrom(e api.OneOf[editions]) (targets.Editions, error) {
	named, ok := e.Value().(namedCuts)
	if !ok {
		return targets.AnyCut(), nil
	}
	keys := make([]release.CutKey, len(named.Cuts))
	for i, k := range named.Cuts {
		keys[i] = release.CutKey(k)
	}
	list, err := targets.NamedCuts(keys...)
	if err != nil {
		return targets.Editions{}, answer(err)
	}
	return list, nil
}

// Profile is exported so the document carries the fields a Profile's
// detail embeds.
type Profile struct {
	ID          string              `json:"id" format:"uuid"`
	Name        string              `json:"name"`
	ContentType string              `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Default     bool                `json:"default" doc:"Whether a Work added with no Profile takes it"`
	Targets     []entry             `json:"targets" nullable:"false" doc:"In Profile order: a file is named after the first it fills"`
	Editions    api.OneOf[editions] `json:"editions"`
	Works       int                 `json:"works" minimum:"0" doc:"The Works wanting from it"`
}

func profileOf(p targets.Profile) Profile {
	out := Profile{
		ID: p.ID.String(), Name: p.Name, ContentType: string(p.ContentType), Default: p.Default, Targets: []entry{},
		Editions: editionsOf(p.Editions), Works: p.Works,
	}
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
	Name        string              `json:"name" minLength:"1" maxLength:"100"`
	ContentType string              `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Targets     []entry             `json:"targets" nullable:"false" uniqueItems:"true" doc:"In Profile order; none is a Profile whose Works want nothing"`
	Editions    api.OneOf[editions] `json:"editions"`
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
	ProfileID   string              `json:"profileId" format:"uuid"`
	ProfileName string              `json:"profileName"`
	Editions    api.OneOf[editions] `json:"editions" doc:"The effective Edition list"`
	From        string              `json:"from" enum:"profile,work" doc:"Whether the Edition list is the Profile's or the Work's own"`
}

type wantsOutput struct{ Body wants }

// profileSpecOf parses a Profile's body.
func profileSpecOf(in newProfile) (targets.ProfileSpec, error) {
	list, err := editionsFrom(in.Editions)
	if err != nil {
		return targets.ProfileSpec{}, err
	}
	s := targets.ProfileSpec{Name: in.Name, ContentType: works.TypeID(in.ContentType), Editions: list}
	for _, e := range in.Targets {
		u, err := storage.ParseUUID(e.TargetID)
		if err != nil {
			return targets.ProfileSpec{}, targetNotFound
		}
		s.Entries = append(s.Entries, targets.Entry{Target: targets.TargetID{UUID: u}, Sourcing: targets.Sourcing(e.Sourcing)})
	}
	return s, nil
}

func (h handlers) createProfile(ctx context.Context, by identity.Admin, in *createProfileInput) (*profileOutput, error) {
	s, err := profileSpecOf(in.Body)
	if err != nil {
		return nil, err
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
	return &wantsOutput{Body: wantsOf(g)}, nil
}

func wantsOf(g targets.Goals) wants {
	out := wants{ProfileID: g.Profile.ID.String(), ProfileName: g.Profile.Name, Editions: editionsOf(g.Editions), From: "profile"}
	if g.Own {
		out.From = "work"
	}
	return out
}
