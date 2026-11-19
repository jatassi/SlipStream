package subtitlesapi

import (
	"context"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/targetsapi"
	"github.com/jatassi/slipstream/internal/works"
)

var stateKinds = map[subtitles.StateKind]string{
	subtitles.NotNeeded: "not_needed", subtitles.Embedded: "embedded", subtitles.Met: "met", subtitles.Missing: "missing",
}

var whys = map[subtitles.NotNeededWhy]string{subtitles.AudioInLanguage: "audio_in_language", subtitles.FullCovers: "full_covers"}

var missingReasons = map[subtitles.MissingReason]string{
	subtitles.NoHolder: "no_holder", subtitles.NoProvider: "no_provider", subtitles.Paused: "paused", subtitles.NoAiredNumber: "no_aired_number",
	subtitles.MapFailed: "map_failed", subtitles.MapDue: "map_due", subtitles.NonePassed: "none_passed", subtitles.NothingFound: "nothing_found",
	subtitles.NeverSearched: "never_searched",
}

type fit struct {
	OffsetMs int64   `json:"offsetMs"`
	Ratio    float64 `json:"ratio" doc:"The framerate change, 1 when none"`
	Segments int     `json:"segments" minimum:"1" doc:"How many stretches of the subtitle have one timing each; more than 1 where the timing jumps, such as at an ad break"`
	Match    int     `json:"match" minimum:"0" maximum:"100" doc:"The timing match, in percent"`
}

type sidecar struct {
	Name         string               `json:"name" doc:"Its name after the video's own, such as en.sdh.srt"`
	Source       string               `json:"source" enum:"provider,upload,found,sibling"`
	Provider     api.Nullable[string] `json:"provider" doc:"The Subtitle Provider it came from"`
	Release      api.Nullable[string] `json:"release" doc:"The release name it was made for, as the provider names it"`
	Final        bool                 `json:"final" doc:"Subtitle search doesn't look for a better one"`
	Fit          api.Nullable[fit]    `json:"fit" doc:"Its Timing sync numbers; null when SlipStream didn't check its timing"`
	TimingDue    bool                 `json:"timingDue" doc:"SlipStream is checking its timing against a new file"`
	Stray        api.Nullable[string] `json:"stray" doc:"Its path under another name while it can't use its own, relative to the root"`
	HindiWarning bool                 `json:"hindiWarning" doc:"A found file named .hi, which Media Servers read as Hindi"`
}

type wantState struct {
	Kind    string                  `json:"kind" enum:"not_needed,embedded,met,missing"`
	Why     api.Nullable[string]    `json:"why" enum:"audio_in_language,full_covers" doc:"Why a not_needed want needs nothing"`
	Track   api.Nullable[int]       `json:"track" doc:"The embedded track that meets it, by its position among the file's subtitle tracks"`
	Sidecar api.Nullable[sidecar]   `json:"sidecar" doc:"The subtitle file that meets it"`
	Missing api.Nullable[string]    `json:"missing" enum:"no_holder,no_provider,paused,no_aired_number,map_failed,map_due,none_passed,nothing_found,never_searched" doc:"Why a missing want has no subtitle yet, the first that applies"`
	Next    api.Nullable[time.Time] `json:"next" doc:"Its next search; null when none is planned"`
}

type lastSearch struct {
	At        time.Time         `json:"at"`
	Found     int               `json:"found" minimum:"0"`
	BestMatch api.Nullable[int] `json:"bestMatch" doc:"The best timing match among them, in percent"`
}

type wantView struct {
	Want   string                    `json:"want" doc:"The want in the path's form: en, pt-BR, en-forced"`
	Wanted targetsapi.WantedLanguage `json:"wanted"`
	State  wantState                 `json:"state"`
	Last   api.Nullable[lastSearch]  `json:"last" doc:"What its last search found; null until one ran"`
}

type track struct {
	Stream   int                  `json:"stream" minimum:"0" doc:"Its position among the file's subtitle tracks"`
	Codec    string               `json:"codec"`
	Language string               `json:"language"`
	Forced   bool                 `json:"forced"`
	Image    bool                 `json:"image"`
	Ignored  bool                 `json:"ignored" doc:"The Admin chose not to count it"`
	Meets    api.Nullable[string] `json:"meets" doc:"The want it meets"`
}

type extra struct {
	Path string `json:"path" doc:"Relative to the root"`
	Name string `json:"name" doc:"Its name after the video's own"`
	Size int64  `json:"size" minimum:"0"`
}

type fileSubtitles struct {
	FileID     string            `json:"fileId" format:"uuid"`
	Path       string            `json:"path" doc:"Relative to the root"`
	Season     api.Nullable[int] `json:"season" doc:"An episode file's Aired season"`
	Episodes   []int             `json:"episodes" nullable:"false" doc:"An episode file's Aired numbers"`
	Wants      []wantView        `json:"wants" nullable:"false" doc:"In Profile order"`
	Embedded   []track           `json:"embedded" nullable:"false" doc:"The media record's subtitle tracks, in stream order"`
	Companions []extra           `json:"companions" nullable:"false" doc:"Subtitle files beside the video that meet no want"`
}

type workSubtitles struct {
	Files []fileSubtitles `json:"files" nullable:"false"`
}

type workSubtitlesOutput struct{ Body workSubtitles }

type workInput struct {
	WorkID string `path:"workId" format:"uuid"`
}

type trackInput struct {
	FileID string `path:"fileId" format:"uuid"`
	Stream int    `path:"stream" minimum:"0"`
}

func sidecarOf(s subtitles.Sidecar) sidecar {
	out := sidecar{Name: s.TailText, Source: string(s.Source), Final: s.Final, TimingDue: s.TimingDue, Stray: some(s.Stray),
		HindiWarning: s.Source == subtitles.Found && s.Tail.Language.String() == "hi"}
	if o := s.Offer; o != nil {
		out.Provider, out.Release = some(string(o.Provider)), some(o.Release)
	}
	if f := s.Fit; f != nil {
		out.Fit = api.Some(fit{OffsetMs: f.Offset.Milliseconds(), Ratio: f.Ratio, Segments: f.Segments, Match: f.Match})
	}
	return out
}

func wantViewOf(w subtitles.WantView) wantView {
	want := subtitles.Want{Language: w.Wanted.Language, Forced: w.Wanted.Forced}
	st := w.State
	out := wantView{Want: want.String(), Wanted: targetsapi.WantedLanguageOf(w.Wanted), State: wantState{Kind: stateKinds[st.Kind]}}
	switch st.Kind {
	case subtitles.NotNeeded:
		out.State.Why = api.Some(whys[st.Why])
	case subtitles.Embedded:
		out.State.Track = api.Some(st.Track)
	case subtitles.Met:
		out.State.Sidecar = api.Some(sidecarOf(*st.Sidecar))
	case subtitles.Missing:
		out.State.Missing = api.Some(missingReasons[st.Missing])
	}
	if !st.Next.IsZero() {
		out.State.Next = api.Some(st.Next)
	}
	if l := w.Last; l != nil {
		last := lastSearch{At: l.At, Found: l.Found}
		if l.BestMatch != nil {
			last.BestMatch = api.Some(*l.BestMatch)
		}
		out.Last = api.Some(last)
	}
	return out
}

func fileOf(f subtitles.FileView) fileSubtitles {
	out := fileSubtitles{FileID: f.File.String(), Path: f.Path, Episodes: []int{}, Wants: []wantView{}, Embedded: []track{}, Companions: []extra{}}
	if f.Season > 0 {
		out.Season = api.Some(f.Season)
		out.Episodes = append(out.Episodes, f.Episodes...)
	}
	for _, w := range f.Wants {
		out.Wants = append(out.Wants, wantViewOf(w))
	}
	for _, t := range f.Embedded {
		tr := track{Stream: t.Ordinal, Codec: t.Codec, Language: string(t.Language), Forced: t.Forced, Image: t.Image, Ignored: t.Ignored}
		if t.Meets != nil {
			tr.Meets = api.Some(t.Meets.String())
		}
		out.Embedded = append(out.Embedded, tr)
	}
	for _, e := range f.Companions {
		out.Companions = append(out.Companions, extra{Path: e.Path, Name: e.Tail, Size: e.Size})
	}
	return out
}

func (h handlers) workSubtitles(ctx context.Context, by identity.Admin, in *workInput) (*workSubtitlesOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	v, err := open(h.o).WorkSubtitles(ctx, by, works.WorkID{UUID: u})
	if err != nil {
		return nil, err
	}
	out := workSubtitles{Files: make([]fileSubtitles, len(v.Files))}
	for i, f := range v.Files {
		out.Files[i] = fileOf(f)
	}
	return &workSubtitlesOutput{Body: out}, nil
}

func (h handlers) ignore(ignored bool) func(ctx context.Context, by identity.Admin, in *trackInput) (*struct{}, error) {
	return func(ctx context.Context, by identity.Admin, in *trackInput) (*struct{}, error) {
		f, err := fileID(in.FileID)
		if err != nil {
			return nil, err
		}
		return nil, answer(open(h.o).IgnoreTrack(ctx, by, f, in.Stream, ignored))
	}
}

var workNotFound = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")
