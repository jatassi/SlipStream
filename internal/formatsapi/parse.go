package formatsapi

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabsapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type draft struct {
	FormatID            string      `json:"formatId,omitempty" format:"uuid" doc:"The saved Custom Format the draft stands in for, whose app it is written in; left out, the draft is a new format in SlipStream's own values"`
	Name                string      `json:"name" minLength:"1" maxLength:"100"`
	IncludeWhenRenaming bool        `json:"includeWhenRenaming,omitempty"`
	Conditions          []condition `json:"conditions" nullable:"false"`
}

type parse struct {
	Names  []string `json:"names" minItems:"1" maxItems:"100" doc:"Release names, or file paths when kind is file"`
	WorkID string   `json:"workId,omitempty" format:"uuid" doc:"The Work to judge every name against; left out, each name is matched to the Works whose titles it gives, read as each Content Type in turn"`
	Draft  *draft   `json:"draft,omitempty" doc:"A Custom Format matched as if saved, in place of formatId's"`
	Kind   string   `json:"kind,omitempty" enum:"release,file" default:"release" doc:"How each name is read. release reads it as a release title. file reads it as a file path, and reads its folder's name when the file name gives no title."`
}

type parseInput struct{ Body parse }

type readFacts struct {
	Quality    *string           `json:"quality" doc:"The Quality's name, such as WEBDL-1080p; null when the name gives none"`
	Source     *string           `json:"source" doc:"The fine source in words, such as WEB-DL"`
	Resolution *string           `json:"resolution"`
	Modifier   *string           `json:"modifier" doc:"The quality modifier in words, such as Remux"`
	Revision   *string           `json:"revision" doc:"The revision in words, such as v2 REPACK; null for the original release"`
	Group      *string           `json:"group" doc:"The release group"`
	Cut        string            `json:"cut" doc:"The cut's key, standard when the name names none"`
	Edition    *string           `json:"edition" doc:"The edition text as the name gives it"`
	HDR        []string          `json:"hdr" nullable:"false" doc:"In words, such as Dolby Vision, best first"`
	Video      *string           `json:"video" doc:"The video codec"`
	Audio      []string          `json:"audio" nullable:"false" doc:"Audio codecs, best first"`
	Channels   []string          `json:"channels" nullable:"false" doc:"Channel layouts, best first"`
	Languages  []string          `json:"languages" nullable:"false" doc:"In words, such as French"`
	IDs        map[string]string `json:"ids" nullable:"false" doc:"The id tags the name carries, by Torznab's names: tmdbid, imdbid, tvdbid"`
	Titles     []string          `json:"titles" nullable:"false" doc:"Every title the name gives, as in A AKA B"`
	Year       *int              `json:"year"`
	Marks      []string          `json:"marks" nullable:"false" doc:"What the name numbers, such as S01E02, in words"`
	Span       string            `json:"span" enum:",one,several,level,levels" doc:"One Item, several Items of one level, a whole level or several levels; empty when the name numbers nothing"`
}

type workRef struct {
	ID    string `json:"id" format:"uuid"`
	Label string `json:"label"`
}

type evaluated struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Negate   bool   `json:"negate"`
	Required bool   `json:"required"`
	Matched  bool   `json:"matched" doc:"After negate"`
	TimedOut bool   `json:"timedOut" doc:"Its pattern ran past the match limit, which counts as not matching"`
}

type explained struct {
	ID         string      `json:"id" format:"uuid"`
	Name       string      `json:"name"`
	Draft      bool        `json:"draft" doc:"It is the request's draft"`
	Matched    bool        `json:"matched"`
	TimedOut   bool        `json:"timedOut"`
	Conditions []evaluated `json:"conditions" nullable:"false"`
}

type triedTarget struct {
	TargetID  string                  `json:"targetId" format:"uuid"`
	Name      string                  `json:"name"`
	Score     int                     `json:"score"`
	WouldFill bool                    `json:"wouldFill" doc:"An automatic search would grab it for this Target"`
	Reasons   []grabsapi.WordedReason `json:"reasons" nullable:"false" doc:"The name's own refusals, then the reasons of the first cell it would fill, else of the first cell it was judged at"`
}

type parsed struct {
	Name     string                `json:"name"`
	Sentence string                `json:"sentence" doc:"What the name reads as, the Works it matches and the Targets it would fill, or why it would fill none"`
	Words    string                `json:"words" doc:"The facts in words, joined with a middle dot: Quality, the revision when it isn't the original, HDR, audio and channels, release group and size"`
	Facts    readFacts             `json:"facts"`
	Work     api.Nullable[workRef] `json:"work" doc:"The one Work the name is judged against: the given Work, or the only Work it matches; null when it matches several or none"`
	Works    []workRef             `json:"works" nullable:"false" doc:"The given Work when the name matches it, else every Work whose title and year it gives"`
	Covers   []string              `json:"covers" nullable:"false" format:"uuid" doc:"The Items the name covers of the one Work it was judged against"`
	Formats  []explained           `json:"formats" nullable:"false" doc:"Every Custom Format, the draft included, by name"`
	Targets  []triedTarget         `json:"targets" nullable:"false" doc:"With one Work, its Profile's Targets judged against its held files; with several or none, every Target with no held file"`
}

type parseResults struct {
	Results []parsed `json:"results" nullable:"false" doc:"In the order of names"`
}

type parseOutput struct{ Body parseResults }

func (h handlers) parseReleases(ctx context.Context, by identity.Admin, in *parseInput) (*parseOutput, error) {
	trial := grabs.Trial{Names: in.Body.Names, File: in.Body.Kind == "file"}
	if in.Body.WorkID != "" {
		u, err := storage.ParseUUID(in.Body.WorkID)
		if err != nil {
			return nil, workNotFound
		}
		trial.Work = &works.WorkID{UUID: u}
	}
	if d := in.Body.Draft; d != nil {
		f, err := h.draftOf(ctx, by, *d)
		if err != nil {
			return nil, err
		}
		trial.Draft = &f
	}
	tried, err := grabber(h.goals).Try(ctx, by, trial)
	switch {
	case errors.Is(err, works.ErrNoWork):
		return nil, workNotFound
	case err != nil && trial.Draft != nil:
		return nil, formatRefused(*trial.Draft, err, "draft.", "try it")
	case err != nil:
		return nil, err
	}
	out := parseResults{Results: make([]parsed, len(tried))}
	for i, t := range tried {
		out.Results[i] = parsedOf(t, trial.Draft)
	}
	return &parseOutput{Body: out}, nil
}

// draftOf is d as the format it stands in for: in that format's app and
// under its id, or new in SlipStream's own values.
func (h handlers) draftOf(ctx context.Context, by identity.Admin, d draft) (formats.Format, error) {
	f := newFormat{Name: d.Name, IncludeWhenRenaming: d.IncludeWhenRenaming, Conditions: d.Conditions}.format()
	f.Dialect = formats.SlipStream
	if d.FormatID == "" {
		return f, nil
	}
	id, err := formatID(d.FormatID)
	if err != nil {
		return formats.Format{}, err
	}
	saved, err := open(h.goals).GetFormat(ctx, by, id)
	if err != nil {
		return formats.Format{}, notFound(err)
	}
	f.ID, f.Dialect = id, saved.Dialect
	return f, nil
}

func parsedOf(t grabs.Tried, d *formats.Format) parsed {
	out := parsed{
		Name: t.Name, Sentence: t.Sentence, Words: release.Describe(t.Read), Facts: factsOf(t.Read),
		Works: make([]workRef, len(t.Works)), Covers: make([]string, len(t.Covers)),
		Formats: make([]explained, len(t.Formats)), Targets: make([]triedTarget, len(t.Targets)),
	}
	if w := t.Work; w != nil {
		out.Work = api.Some(workRef{ID: w.ID.String(), Label: w.Label})
	}
	for i, w := range t.Works {
		out.Works[i] = workRef{ID: w.ID.String(), Label: w.Label}
	}
	for i, it := range t.Covers {
		out.Covers[i] = it.String()
	}
	for i, e := range t.Formats {
		x := explained{ID: e.Format.String(), Name: e.Name, Draft: d != nil && e.Format == d.ID, Matched: e.Matched, TimedOut: e.TimedOut, Conditions: make([]evaluated, len(e.Conditions))}
		for j, c := range e.Conditions {
			x.Conditions[j] = evaluated{Name: c.Name, Kind: string(c.Kind), Negate: c.Negate, Required: c.Required, Matched: c.Matched, TimedOut: c.TimedOut}
		}
		out.Formats[i] = x
	}
	for i, tt := range t.Targets {
		out.Targets[i] = triedTarget{TargetID: tt.Target.ID.String(), Name: tt.Target.Name, Score: tt.Score, WouldFill: tt.WouldFill, Reasons: grabsapi.WordedOf(tt.Reasons)}
	}
	return out
}

func factsOf(rd release.Read) readFacts {
	f, n := rd.Facts, rd.Name
	out := readFacts{
		Source: wordsOf(formats.Source, f.Source), Resolution: wordsOf(formats.Resolution, f.Resolution), Modifier: wordsOf(formats.QualityModifier, f.Modifier),
		Group: stringOf(f.Group), Cut: string(f.Cut), Edition: stringOf(f.Edition),
		HDR: listOf(f.HDR, release.HDR.Words), Video: stringOf(f.Video), Audio: listOf(f.Audio, plain[release.AudioCodec]), Channels: listOf(f.Channels, plain[release.Channels]),
		Languages: listOf(f.Languages, func(l release.Language) string { return formats.Words(formats.Language, string(l)) }),
		IDs:       map[string]string{}, Titles: append([]string{}, n.Titles...), Marks: []string{}, Span: n.Span.String(),
	}
	if q, ok := f.Quality.Get(); ok {
		name := q.Info().Name
		out.Quality = &name
	}
	if !f.Revision.Original() {
		r := f.Revision.String()
		out.Revision = &r
	}
	for k, v := range f.IDs {
		out.IDs[k] = v
	}
	if y, ok := n.Year.Get(); ok {
		out.Year = &y
	}
	for _, m := range n.Marks {
		out.Marks = append(out.Marks, m.Words)
	}
	return out
}

// wordsOf is f's value in the words of k's SlipStream values.
func wordsOf[T interface{ String() string }](k formats.Kind, f release.Fact[T]) *string {
	v, ok := f.Get()
	if !ok {
		return nil
	}
	s := formats.Words(k, v.String())
	return &s
}

func stringOf[T ~string](f release.Fact[T]) *string {
	v, ok := f.Get()
	if !ok {
		return nil
	}
	s := string(v)
	return &s
}

func listOf[T any](f release.Fact[[]T], words func(T) string) []string {
	vs, _ := f.Get()
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = words(v)
	}
	return out
}

func plain[T ~string](v T) string { return string(v) }
