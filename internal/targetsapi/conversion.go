package targetsapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// ConversionSettings is a Target's Conversion settings on the wire.
type ConversionSettings struct {
	Codec          string `json:"codec" enum:"h264,hevc,av1"`
	MaxHeight      int    `json:"maxHeight" enum:"2160,1080,720,480" doc:"The largest output height. A wider picture fits the 16:9 box of that height"`
	Kbps           int    `json:"kbps" minimum:"100" maximum:"200000" doc:"The video bitrate at maxHeight"`
	Depth          int    `json:"depth" enum:"8,10"`
	Range          string `json:"range" enum:"sdr,hdr10,source" doc:"source keeps the source's range, with HDR10+ written as HDR10"`
	KeepDV         bool   `json:"keepDolbyVision"`
	Curve          string `json:"curve" enum:"bt2390,hable,mobius,reinhard" doc:"The tone mapping curve from HDR to SDR"`
	Audio          string `json:"audio" enum:"copy,stereo_aac,ac3_51_and_stereo_aac"`
	DropCommentary bool   `json:"dropCommentary"`
	Container      string `json:"container" enum:"mkv,mp4"`
}

type conversionSpec struct {
	ConversionSettings
	RunOnNodeID *string `json:"runOnNodeId" format:"uuid" doc:"The one Node its Conversions run on; null for any Node that can"`
}

type cause struct {
	Why      string              `json:"why" enum:"no_role,lapsed,runs_elsewhere,pinned_elsewhere,failed_here,untested,test_failed,retesting,proof_lost,processor_off,none_online,no_holder"`
	Class    string              `json:"class" enum:"h264_8,hevc_8,hevc_10,av1_8,av1_10,hdr10,hlg,dv5,dv81,decode_sdr,decode_hdr10" doc:"The source class it covers"`
	Shape    string              `json:"shape" doc:"The settings shape it covers, such as h264.8.sdr.bt2390"`
	Count    int                 `json:"count" minimum:"1" doc:"The Conversions waiting on it"`
	Node     api.Nullable[named] `json:"node" doc:"The Node its sentence names; null when none"`
	Notified time.Time           `json:"notifiedAt" doc:"When the Admin was notified"`
}

type named struct {
	ID   string `json:"id" format:"uuid"`
	Name string `json:"name"`
}

type conversionView struct {
	ConversionSettings
	RunOnNodeID *string `json:"runOnNodeId" format:"uuid" doc:"null for any Node that can"`
	Version     int     `json:"version" minimum:"1" doc:"Rises with each change of the settings, not of Run on"`
	Older       int     `json:"older" minimum:"0" doc:"Made files the Target's older settings made"`
	Causes      []cause `json:"causes" nullable:"false" doc:"Why no Node can convert some of its Conversions"`
	DefaultKbps int     `json:"defaultKbps" minimum:"100" doc:"The bitrate a new setting starts at"`
}

// Problem is one reason the Target's Conversion settings can't be saved,
// worded by the web from its code and facts.
type Problem struct {
	Code  string            `json:"code" enum:"codec_filtered,range_from_sdr,outside_size,container_audio,quality_unranked"`
	Field string            `json:"field" doc:"The Conversion setting it is about, such as codec"`
	Facts map[string]string `json:"facts" doc:"What its words name, by key"`
}

type sourceInUse struct {
	Profiles []string `json:"profiles" nullable:"false" doc:"The Profiles that make another Target's files from it"`
}

var (
	conversionRefused = api.NewCarrying[[]Problem](http.StatusUnprocessableEntity, "conversion_refused", "These Conversion settings can't make a file this Target accepts", "problems")
	targetIsASource   = api.NewCarryingMembers[sourceInUse](http.StatusConflict, "target_is_a_source", "Profiles make other Targets' files from this Target's files")
)

type madeRow struct {
	WorkID  string    `json:"workId" format:"uuid"`
	Title   string    `json:"title"`
	Files   int       `json:"files" minimum:"0"`
	Seconds int64     `json:"seconds" minimum:"0"`
	Bytes   int64     `json:"bytes" minimum:"0"`
	AddedAt time.Time `json:"addedAt"`
}

type madeImpact struct {
	Files         int       `json:"files" minimum:"0" doc:"Files the save has SlipStream make"`
	Works         int       `json:"works" minimum:"0"`
	Seconds       int64     `json:"seconds" minimum:"0" doc:"About how long they take to make"`
	Bytes         int64     `json:"bytes" minimum:"0" doc:"About how much they take on disk"`
	Nodes         []string  `json:"nodes" nullable:"false" doc:"The Nodes forecast to make them"`
	Unconvertible []string  `json:"unconvertible" nullable:"false" format:"uuid" doc:"The Works no Node can convert today"`
	Matching      int       `json:"matching" minimum:"0" doc:"The Works the backfill's rest moves to a Profile that matches without the made-from Target"`
	Rows          []madeRow `json:"rows" nullable:"false"`
}

type remakeImpact struct {
	Older int       `json:"older" minimum:"0" doc:"Made files the Target's older settings made"`
	Rows  []madeRow `json:"rows" nullable:"false"`
}

type backfillRest struct {
	ProfileID string `json:"profileId,omitempty" format:"uuid" doc:"An existing Profile the rest move to"`
	NewName   string `json:"newName,omitempty" maxLength:"100" doc:"The name of a new Profile, the old one without the made-from Target, that the rest move to"`
}

type backfill struct {
	Convert []string     `json:"convert" nullable:"false" format:"uuid" doc:"The Works to convert"`
	Rest    backfillRest `json:"rest" doc:"Where the other Works go"`
	Default string       `json:"default" format:"uuid" doc:"The Profile new Works take"`
}

// settingsOf is s on the wire.
func settingsOf(s encode.Settings) ConversionSettings {
	return ConversionSettings{
		Codec: string(s.Codec), MaxHeight: int(s.MaxHeight), Kbps: s.Kbps, Depth: int(s.Depth), Range: string(s.Range), KeepDV: s.KeepDV,
		Curve: string(s.Curve), Audio: string(s.Audio), DropCommentary: s.DropCommentary, Container: string(s.Container),
	}
}

// conversionOf parses a body's Conversion settings.
func conversionOf(in *conversionSpec) (*targets.Conversion, error) {
	if in == nil {
		return nil, nil
	}
	c := in.ConversionSettings
	s, err := encode.Parse(encode.Raw{
		Codec: c.Codec, MaxHeight: c.MaxHeight, Kbps: c.Kbps, Depth: c.Depth, Range: c.Range, KeepDV: c.KeepDV, Curve: c.Curve,
		Audio: c.Audio, DropCommentary: c.DropCommentary, Container: c.Container,
	})
	if fe, ok := errors.AsType[*encode.FieldError](err); ok {
		return nil, api.Invalid.On("conversion."+wireField(fe.Field), settingWords[fe.Field])
	}
	out := &targets.Conversion{Settings: s}
	if in.RunOnNodeID != nil {
		u, err := storage.ParseUUID(*in.RunOnNodeID)
		if err != nil {
			return nil, runOnRefused
		}
		out.RunOn = &role.NodeID{UUID: u}
	}
	return out, nil
}

var runOnRefused = api.Invalid.On("conversion.runOnNodeId", "That Node doesn't have the Converting Role. Choose a Node with the Role, or any Node.")

// settingWords is the sentence of each setting encode.Parse refuses past
// what the document states.
var settingWords = map[string]string{
	"Depth":  "10 bit needs HEVC or AV1. Choose one of them, or 8 bit.",
	"KeepDV": "Keeping Dolby Vision needs HEVC or AV1 and a range other than SDR.",
}

// wireField is a setting's encode.Raw name as the wire spells it.
func wireField(name string) string {
	if name == "KeepDV" {
		return "keepDolbyVision"
	}
	first, size := utf8.DecodeRuneInString(name)
	return string(unicode.ToLower(first)) + name[size:]
}

// checkRunOn refuses Conversion settings whose Run on names a Node that
// doesn't hold the Converting Role.
func checkRunOn(g Goals, c *targets.Conversion) error {
	if c == nil || c.RunOn == nil || roster(g).Holds(role.Converting, *c.RunOn) {
		return nil
	}
	return runOnRefused
}

func problemsOf(ps []targets.Problem) []Problem {
	out := make([]Problem, len(ps))
	for i, p := range ps {
		out[i] = Problem{Code: string(p.Code), Field: wireField(p.Field), Facts: p.Facts}
	}
	return out
}

// made is what a Target's editor shows of its Conversions: the open
// causes of each Target, the made files its older settings made and the
// Nodes' names.
type made struct {
	causes map[targets.TargetID][]conversions.Cause
	older  map[targets.TargetID]int
	names  map[nodes.ID]nodes.Named
}

func (h handlers) madeOf(ctx context.Context, ts []targets.Target) (made, error) {
	out := made{older: map[targets.TargetID]int{}}
	var ids []targets.TargetID
	for _, t := range ts {
		if t.Conversion != nil {
			ids = append(ids, t.ID)
		}
	}
	if len(ids) == 0 {
		return out, nil
	}
	c := converts(h.goals)
	var err error
	if out.causes, err = c.Causes(ctx, ids); err != nil {
		return made{}, err
	}
	for _, t := range ts {
		if t.Conversion != nil {
			if out.older[t.ID], err = c.Older(ctx, t.ID, t.Conversion.Version); err != nil {
				return made{}, err
			}
		}
	}
	out.names, err = roster(h.goals).Names(ctx)
	return out, err
}

// defaultKbps is the bitrate Conversion settings of t start at, made at
// height from a WEB-DL source.
func defaultKbps(t targets.Target, height encode.Height) int {
	return targets.DefaultKbps(t, height, release.OriginWEBDL)
}

func (m made) viewOf(t targets.Target) api.Nullable[conversionView] {
	c := t.Conversion
	if c == nil {
		return api.Nullable[conversionView]{}
	}
	out := conversionView{
		ConversionSettings: settingsOf(c.Settings), Version: c.Version, Older: m.older[t.ID], Causes: []cause{},
		DefaultKbps: defaultKbps(t, c.Settings.MaxHeight),
	}
	if c.RunOn != nil {
		out.RunOnNodeID = new(c.RunOn.String())
	}
	for _, k := range m.causes[t.ID] {
		row := cause{Why: string(k.Key.Why), Class: string(k.Key.Class), Shape: k.Key.Shape, Count: k.Count, Notified: k.Notified}
		if k.Node != nil {
			row.Node = api.Some(named{ID: k.Node.String(), Name: m.names[*k.Node].Name})
		}
		out.Causes = append(out.Causes, row)
	}
	return api.Some(out)
}

func madeRowsOf(rows []files.MadeRow) []madeRow {
	out := make([]madeRow, len(rows))
	for i, r := range rows {
		out[i] = madeRow{WorkID: r.Work.String(), Files: r.Files, Seconds: r.Seconds, Bytes: r.Bytes, AddedAt: r.Added}
	}
	return out
}

func madeOf(m files.MadeImpact) madeImpact {
	out := madeImpact{
		Files: m.Files, Works: m.Works, Seconds: m.Seconds, Bytes: m.Bytes, Nodes: append([]string{}, m.Nodes...),
		Unconvertible: []string{}, Matching: m.Matching, Rows: madeRowsOf(m.Rows),
	}
	for _, w := range m.Unconvertible {
		out.Unconvertible = append(out.Unconvertible, w.String())
	}
	return out
}

func remakeOf(r files.RemakeImpact) remakeImpact {
	return remakeImpact{Older: r.Older, Rows: madeRowsOf(r.Rows)}
}

// titled names the Works of i's made and remade rows.
func (h handlers) titled(ctx context.Context, i *Impact) error {
	var ws []works.WorkID
	for _, rows := range [][]madeRow{i.Made.Rows, i.Remake.Rows} {
		for _, r := range rows {
			u, _ := storage.ParseUUID(r.WorkID)
			ws = append(ws, works.WorkID{UUID: u})
		}
	}
	if len(ws) == 0 {
		return nil
	}
	titles, err := converts(h.goals).Titles(ctx, ws)
	if err != nil {
		return err
	}
	for _, rows := range [][]madeRow{i.Made.Rows, i.Remake.Rows} {
		for k := range rows {
			u, _ := storage.ParseUUID(rows[k].WorkID)
			rows[k].Title = titles[works.WorkID{UUID: u}]
		}
	}
	return nil
}

// sourcingWords is the sentence of each way a Profile row's sourcing is
// refused.
var sourcingWords = []struct {
	err   error
	words string
}{
	{targets.ErrBadSourcing, "A made-from Target needs the Target it's made from, and only a Downloaded, or made from Target takes a wait."},
	{targets.ErrBadGrace, "Wait between 1 hour and 90 days for a download."},
	{targets.ErrNoConversion, "Turn on Conversion settings for a Target before a Profile makes its files from another Target's."},
	{targets.ErrSourceNotAbove, "Make a Target's files from a downloaded Target above it in the Profile."},
}

// conversionAnswer is the problem of err, a refused Conversion or sourcing save,
// nil for any other error.
func conversionAnswer(err error) error {
	if refused, ok := errors.AsType[*targets.ConversionRefusedError](err); ok {
		return conversionRefused.With(problemsOf(refused.Problems))
	}
	if inUse, ok := errors.AsType[*targets.SourceInUseError](err); ok {
		words := " depends on this Target to make files from other Targets' files. Change its made-from Targets first."
		if len(inUse.Profiles) > 1 {
			words = " depend on this Target to make files from other Targets' files. Change their made-from Targets first."
		}
		return targetIsASource.Explained(joinNames(inUse.Profiles)+words, sourceInUse{Profiles: inUse.Profiles})
	}
	for _, s := range sourcingWords {
		if errors.Is(err, s.err) {
			return api.Invalid.On("targets", s.words)
		}
	}
	switch {
	case errors.Is(err, targets.ErrNotOnProfile):
		return api.Invalid.On("backfill.convert", "Choose only Works that use this Profile.")
	case errors.Is(err, targets.ErrRestStays):
		return api.Invalid.On("backfill.rest", "Move the other Works to a Profile other than this one.")
	}
	return nil
}

func joinNames(names []string) string {
	if len(names) == 1 {
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
