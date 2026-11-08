package targetsapi

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

type size struct {
	Quality string   `json:"quality" enum:"sdtv,dvd,webrip-480p,hdtv-720p,webrip-720p,webdl-720p,bluray-720p,hdtv-1080p,webrip-1080p,webdl-1080p,bluray-1080p,remux-1080p,hdtv-2160p,webrip-2160p,webdl-2160p,bluray-2160p,remux-2160p,cam"`
	Min     float64  `json:"min" minimum:"0" doc:"MiB per minute of the runtime the file covers"`
	Max     *float64 `json:"max" minimum:"0" doc:"MiB per minute; null is no limit"`
}

type filter struct {
	Attribute string   `json:"attribute" enum:"hdr,video_codec,audio_codec,channels,language"`
	Mode      string   `json:"mode" enum:"only,not" doc:"Only passes a fact holding one of the values, not one holding none; an unknown fact passes"`
	Values    []string `json:"values" nullable:"false" minItems:"1" uniqueItems:"true" doc:"The attribute's values by name; a language is an ISO 639 code or original"`
}

type formatScore struct {
	FormatID string `json:"formatId" format:"uuid"`
	Score    int    `json:"score" doc:"Added to a Release's score when it matches the Custom Format"`
}

type scores struct {
	Formats      []formatScore `json:"formats" nullable:"false" doc:"Every Custom Format the Target scores; a format left out scores 0"`
	Minimum      int           `json:"minimum" doc:"A Release scoring below it isn't grabbed"`
	UpgradeUntil int           `json:"upgradeUntil" doc:"A file at or past the cutoff upgrades to better-scored Releases until it scores this"`
	Increment    int           `json:"increment" minimum:"1" doc:"How far a Release's score must pass the file's to replace it in the same tier"`
}

type newScores struct {
	Formats      []formatScore `json:"formats,omitempty" doc:"A Custom Format left out scores 0"`
	Minimum      int           `json:"minimum,omitempty" doc:"A Release scoring below it isn't grabbed"`
	UpgradeUntil int           `json:"upgradeUntil,omitempty" doc:"A file at or past the cutoff upgrades to better-scored Releases until it scores this"`
	Increment    int           `json:"increment,omitempty" minimum:"1" default:"1" doc:"How far a Release's score must pass the file's to replace it in the same tier"`
}

type profileRef struct {
	ID   string `json:"id" format:"uuid"`
	Name string `json:"name"`
}

type target struct {
	ID        string       `json:"id" format:"uuid"`
	Name      string       `json:"name"`
	Tiers     [][]string   `json:"tiers" nullable:"false" doc:"Quality keys, best tier first; a tier's qualities rank as equal"`
	Cutoff    string       `json:"cutoff" doc:"The first quality of the tier at which a file is final"`
	Filters   []filter     `json:"filters" nullable:"false" doc:"Every attribute that isn't Any, in attribute order"`
	Sizes     []size       `json:"sizes" nullable:"false" doc:"Every accepted quality's size, in tier order"`
	Scores    scores       `json:"scores"`
	Propers   string       `json:"propers" enum:"prefer_and_upgrade,prefer_no_upgrade,dont_prefer" doc:"prefer_and_upgrade ranks propers and repacks higher and has one replace a file of its Quality even past the cutoff. prefer_no_upgrade only ranks them higher. dont_prefer leaves them to the scores"`
	Profiles  []profileRef `json:"profiles" nullable:"false" doc:"The Profiles holding it, by name"`
	Contains  []string     `json:"contains" nullable:"false" format:"uuid" doc:"The other Targets every file of which also meets this one"`
	CreatedAt time.Time    `json:"createdAt"`
}

func targetOf(t targets.Target) target {
	s := t.Spec()
	out := target{
		ID: t.ID.String(), Name: t.Name, Tiers: keysOf(s.Tiers), Cutoff: s.Cutoff.String(), Filters: []filter{}, Sizes: []size{},
		Scores: scores{
			Formats: []formatScore{}, Minimum: t.Scores.Minimum, UpgradeUntil: t.Scores.UpgradeUntil, Increment: t.Scores.Increment,
		},
		Propers: t.Propers.String(), Profiles: []profileRef{}, Contains: []string{}, CreatedAt: t.CreatedAt.Time(),
	}
	for id, score := range t.Scores.Formats {
		out.Scores.Formats = append(out.Scores.Formats, formatScore{FormatID: id.String(), Score: score})
	}
	slices.SortFunc(out.Scores.Formats, func(a, b formatScore) int { return strings.Compare(a.FormatID, b.FormatID) })
	for _, tier := range s.Tiers {
		for _, q := range tier {
			out.Sizes = append(out.Sizes, size{Quality: q.String(), Min: s.Sizes[q].Min, Max: s.Sizes[q].Max})
		}
	}
	for _, a := range targets.Attributes() {
		if mode, values := t.Filters.Of(a); mode != targets.Any {
			out.Filters = append(out.Filters, filter{Attribute: string(a), Mode: string(mode), Values: values})
		}
	}
	for _, p := range t.Profiles {
		out.Profiles = append(out.Profiles, profileRef{ID: p.ID.String(), Name: p.Name})
	}
	for _, id := range t.Contains {
		out.Contains = append(out.Contains, id.String())
	}
	return out
}

type newTarget struct {
	Name    string     `json:"name" minLength:"1" maxLength:"100"`
	Tiers   [][]string `json:"tiers" nullable:"false" minItems:"1" doc:"Quality keys, best tier first; a tier's qualities rank as equal"`
	Cutoff  string     `json:"cutoff" enum:"sdtv,dvd,webrip-480p,hdtv-720p,webrip-720p,webdl-720p,bluray-720p,hdtv-1080p,webrip-1080p,webdl-1080p,bluray-1080p,remux-1080p,hdtv-2160p,webrip-2160p,webdl-2160p,bluray-2160p,remux-2160p,cam" doc:"Any quality of the tier at which a file is final"`
	Filters []filter   `json:"filters,omitempty" doc:"An attribute left out is Any"`
	Sizes   []size     `json:"sizes,omitempty" doc:"An accepted quality left out takes its floor and no maximum"`
	Scores  newScores  `json:"scores,omitempty" doc:"Left out, every score is 0 and the increment is 1"`
	Propers string     `json:"propers,omitempty" enum:"prefer_and_upgrade,prefer_no_upgrade,dont_prefer" default:"prefer_and_upgrade" doc:"prefer_and_upgrade ranks propers and repacks higher and has one replace a file of its Quality even past the cutoff. prefer_no_upgrade only ranks them higher. dont_prefer leaves them to the scores"`
}

// specOf parses a Target's body. Checks the document can't state answer
// invalid at the field they name.
func specOf(in newTarget) (targets.Spec, error) {
	s := targets.Spec{Name: in.Name, Sizes: map[release.Quality]targets.Size{}}
	var bad []api.FieldError
	field := func(location string, err error) {
		bad = append(bad, api.FieldError{Location: location, Message: err.Error()})
	}
	for i, tier := range in.Tiers {
		var qs []release.Quality
		for j, key := range tier {
			q, err := release.ParseQuality(key)
			if err != nil {
				field(fmt.Sprintf("body.tiers[%d][%d]", i, j), err)
			}
			qs = append(qs, q)
		}
		s.Tiers = append(s.Tiers, qs)
	}
	cutoff, _ := release.ParseQuality(in.Cutoff)
	s.Cutoff = cutoff
	filtered := map[string]bool{}
	for i, f := range in.Filters {
		mode, _ := targets.ParseMode(f.Mode)
		if filtered[f.Attribute] {
			field(fmt.Sprintf("body.filters[%d].attribute", i), errors.New("the attribute is listed twice"))
		} else if err := s.Filters.Set(targets.Attribute(f.Attribute), mode, f.Values); err != nil {
			field(fmt.Sprintf("body.filters[%d].values", i), err)
		}
		filtered[f.Attribute] = true
	}
	for i, z := range in.Sizes {
		q, _ := release.ParseQuality(z.Quality)
		if _, twice := s.Sizes[q]; twice {
			field(fmt.Sprintf("body.sizes[%d].quality", i), errors.New("the quality is listed twice"))
		}
		s.Sizes[q] = targets.Size{Min: z.Min, Max: z.Max}
	}
	s.Scores = targets.Scores{Formats: map[formats.ID]int{}, Minimum: in.Scores.Minimum, UpgradeUntil: in.Scores.UpgradeUntil, Increment: in.Scores.Increment}
	for i, fs := range in.Scores.Formats {
		id, _ := targets.ParseFormatID(fs.FormatID)
		if _, twice := s.Scores.Formats[id]; twice {
			field(fmt.Sprintf("body.scores.formats[%d].formatId", i), errors.New("the Custom Format is listed twice"))
		}
		s.Scores.Formats[id] = fs.Score
	}
	s.Propers, _ = targets.ParsePropers(in.Propers)
	if len(bad) > 0 {
		return targets.Spec{}, api.Invalid.Fields(bad...)
	}
	return s, nil
}

type createTargetInput struct{ Body newTarget }

type targetInput struct {
	TargetID string `path:"targetId" format:"uuid"`
}

type targetOutput struct{ Body target }

type targetList struct {
	Targets []target `json:"targets" nullable:"false" doc:"By name"`
}

type targetListOutput struct{ Body targetList }

type quality struct {
	Key        string  `json:"key" enum:"sdtv,dvd,webrip-480p,hdtv-720p,webrip-720p,webdl-720p,bluray-720p,hdtv-1080p,webrip-1080p,webdl-1080p,bluray-1080p,remux-1080p,hdtv-2160p,webrip-2160p,webdl-2160p,bluray-2160p,remux-2160p,cam"`
	Name       string  `json:"name"`
	Source     string  `json:"source" enum:"tv,hdtv,dvd,webrip,webdl,bluray,remux,cinema"`
	Resolution *string `json:"resolution" enum:"2160p,1080p,720p,480p" doc:"Null for cinema sources"`
	Cinema     bool    `json:"cinema" doc:"A cinema recording: kept out of a Target unless an Admin adds it"`
	Floor      float64 `json:"floor" minimum:"0" doc:"A new Target's minimum size, in MiB per minute"`
}

type preset struct {
	Key   string     `json:"key" enum:"disc-or-resolution,anything-better,resolution-only"`
	Name  string     `json:"name"`
	Tiers [][]string `json:"tiers" nullable:"false" doc:"Every quality, best tier first. Over chosen qualities, keep these tiers to them; with no upgrades, put the cutoff at the bottom tier"`
}

type qualityList struct {
	Qualities []quality `json:"qualities" nullable:"false" doc:"Low to high, the presets' default order"`
	Presets   []preset  `json:"presets" nullable:"false" doc:"The default first"`
}

type qualityListOutput struct{ Body qualityList }

func keysOf(tiers [][]release.Quality) [][]string {
	out := [][]string{}
	for _, tier := range tiers {
		keys := []string{}
		for _, q := range tier {
			keys = append(keys, q.String())
		}
		out = append(out, keys)
	}
	return out
}

func (h handlers) listQualities(context.Context, identity.Admin, *struct{}) (*qualityListOutput, error) {
	out := qualityList{Qualities: []quality{}, Presets: []preset{}}
	for _, q := range release.Qualities() {
		i := q.Info()
		row := quality{Key: i.Key, Name: i.Name, Source: i.Source.String(), Cinema: i.Cinema(), Floor: i.Floor}
		if !i.Cinema() {
			row.Resolution = new(i.Resolution.String())
		}
		out.Qualities = append(out.Qualities, row)
	}
	for _, p := range targets.Presets() {
		out.Presets = append(out.Presets, preset{Key: string(p), Name: p.Name(), Tiers: keysOf(p.Tiers())})
	}
	return &qualityListOutput{Body: out}, nil
}

func (h handlers) createTarget(ctx context.Context, by identity.Admin, in *createTargetInput) (*targetOutput, error) {
	s, err := specOf(in.Body)
	if err != nil {
		return nil, err
	}
	t, err := open(h.goals).CreateTarget(ctx, by, s)
	if err != nil {
		return nil, answer(err)
	}
	return &targetOutput{Body: targetOf(t)}, nil
}

func (h handlers) listTargets(ctx context.Context, by identity.Admin, _ *struct{}) (*targetListOutput, error) {
	list, err := open(h.goals).ListTargets(ctx, by)
	if err != nil {
		return nil, err
	}
	out := targetList{Targets: []target{}}
	for _, t := range list {
		out.Targets = append(out.Targets, targetOf(t))
	}
	return &targetListOutput{Body: out}, nil
}

func targetID(s string) (targets.TargetID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return targets.TargetID{}, noSuchTarget
	}
	return targets.TargetID{UUID: u}, nil
}

// onPath answers a Target the path names but none has as 404.
func onPath(err error) error {
	if errors.Is(err, targets.ErrNoTarget) {
		return noSuchTarget
	}
	return answer(err)
}

func (h handlers) getTarget(ctx context.Context, by identity.Admin, in *targetInput) (*targetOutput, error) {
	id, err := targetID(in.TargetID)
	if err != nil {
		return nil, err
	}
	t, err := open(h.goals).GetTarget(ctx, by, id)
	if err != nil {
		return nil, onPath(err)
	}
	return &targetOutput{Body: targetOf(t)}, nil
}

func (h handlers) duplicateTarget(ctx context.Context, by identity.Admin, in *targetInput) (*targetOutput, error) {
	id, err := targetID(in.TargetID)
	if err != nil {
		return nil, err
	}
	t, err := open(h.goals).DuplicateTarget(ctx, by, id)
	if err != nil {
		return nil, onPath(err)
	}
	return &targetOutput{Body: targetOf(t)}, nil
}

func (h handlers) deleteTarget(ctx context.Context, by identity.Admin, in *targetInput) (*struct{}, error) {
	id, err := targetID(in.TargetID)
	if err != nil {
		return nil, err
	}
	return nil, onPath(open(h.goals).DeleteTarget(ctx, by, id))
}
