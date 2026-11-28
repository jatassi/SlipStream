package targetsapi

import (
	"context"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type entry struct {
	TargetID     string `json:"targetId" format:"uuid"`
	Sourcing     string `json:"sourcing" enum:"download,derive,download_then_derive" doc:"How the Target gets its file: download, made from another Target's file (derive), or downloaded, or made from another Target's file after a wait (download_then_derive)"`
	FromTargetID string `json:"fromTargetId,omitempty" format:"uuid" doc:"For derive and download_then_derive: the Target above it whose file it is made from"`
	GraceSeconds int    `json:"graceSeconds,omitempty" minimum:"3600" maximum:"7776000" doc:"For download_then_derive: how long SlipStream searches for a download before it makes the file"`
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
	Kind string   `json:"kind" enum:"cuts" doc:"Named cuts: with Standard, each other cut once a release or file of it turns up for an Item; without it, every cut every time"`
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

// WantedLanguage is one of a Profile's Wanted Languages on the wire.
type WantedLanguage struct {
	Language        string `json:"language" minLength:"2" maxLength:"5" doc:"An ISO 639 code, or pt-BR for Brazilian Portuguese"`
	Forced          bool   `json:"forced" doc:"Whether only the lines in another language, such as signs, are wanted, not every line of dialogue"`
	HearingImpaired string `json:"hearingImpaired" enum:"avoid,allow,prefer" doc:"How hearing-impaired subtitles rank. avoid uses one only when nothing else fits, and prefer chooses one first when one fits"`
	Audio           string `json:"audio" enum:"always,no_audio" doc:"When the language is wanted. always is whatever the audio, and no_audio is only when no audio track is in the language"`
}

// WantedLanguageOf is l on the wire.
func WantedLanguageOf(l targets.WantedLanguage) WantedLanguage {
	return WantedLanguage{Language: l.Language.String(), Forced: l.Forced, HearingImpaired: string(l.HI), Audio: string(l.Audio)}
}

// languagesFrom reads a body's Wanted Languages.
func languagesFrom(in []WantedLanguage) ([]targets.WantedLanguage, error) {
	out := make([]targets.WantedLanguage, len(in))
	for i, l := range in {
		c, err := language.Parse(l.Language)
		if err != nil {
			return nil, api.Invalid.On(fmt.Sprintf("languages[%d].language", i), l.Language+" isn't a supported language code. Use an ISO 639 code, such as en, or pt-BR for Brazilian Portuguese.")
		}
		out[i] = targets.WantedLanguage{Language: c, Forced: l.Forced, HI: targets.HI(l.HearingImpaired), Audio: targets.AudioCondition(l.Audio)}
	}
	return out, nil
}

// Profile is exported so the document carries the fields a Profile's
// detail embeds.
type Profile struct {
	ID          string              `json:"id" format:"uuid"`
	Name        string              `json:"name"`
	ContentType string              `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Default     bool                `json:"default" doc:"Whether a Work added with no Profile uses it"`
	Targets     []entry             `json:"targets" nullable:"false" doc:"In Profile order: a file is named after the first Target it is the file for"`
	Editions    api.OneOf[editions] `json:"editions"`
	Languages   []WantedLanguage    `json:"languages" nullable:"false" doc:"The subtitle languages its Works' files want, in the Admin's order"`
	Works       int                 `json:"works" minimum:"0" doc:"The Works that use it"`
}

func profileOf(p targets.Profile) Profile {
	out := Profile{
		ID: p.ID.String(), Name: p.Name, ContentType: string(p.ContentType), Default: p.Default, Targets: []entry{},
		Editions: editionsOf(p.Editions), Languages: []WantedLanguage{}, Works: p.Works,
	}
	for _, l := range p.Languages {
		out.Languages = append(out.Languages, WantedLanguageOf(l))
	}
	for _, e := range p.Entries {
		row := entry{TargetID: e.Target.String(), Sourcing: string(e.Sourcing), GraceSeconds: int(e.Grace / time.Second)}
		if e.From != nil {
			row.FromTargetID = e.From.String()
		}
		out.Targets = append(out.Targets, row)
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
	Targets     []entry             `json:"targets" nullable:"false" uniqueItems:"true" doc:"In Profile order; none is a Profile whose Works need no files"`
	Editions    api.OneOf[editions] `json:"editions"`
	Languages   []WantedLanguage    `json:"languages" nullable:"false" doc:"In the Admin's order, each language once as full and once as forced only at most; none for a Content Type without subtitles"`
	Backfill    *backfill           `json:"backfill,omitempty" doc:"A replace that adds a made-from Target only: which Works to convert, where the rest go, and the default for new Works"`
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
	langs, err := languagesFrom(in.Languages)
	if err != nil {
		return targets.ProfileSpec{}, err
	}
	s := targets.ProfileSpec{Name: in.Name, ContentType: works.TypeID(in.ContentType), Editions: list, Languages: langs}
	for i, e := range in.Targets {
		u, err := storage.ParseUUID(e.TargetID)
		if err != nil {
			return targets.ProfileSpec{}, targetNotFound
		}
		row := targets.Entry{Target: targets.TargetID{UUID: u}, Sourcing: targets.Sourcing(e.Sourcing), Grace: time.Duration(e.GraceSeconds) * time.Second}
		if e.FromTargetID != "" {
			from, err := storage.ParseUUID(e.FromTargetID)
			if err != nil {
				return targets.ProfileSpec{}, api.Invalid.On(fmt.Sprintf("targets[%d].fromTargetId", i), "No Target has this id.")
			}
			row.From = &targets.TargetID{UUID: from}
		}
		s.Entries = append(s.Entries, row)
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
