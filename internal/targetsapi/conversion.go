package targetsapi

import (
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/conversionsapi/notyet"
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
	Notified string              `json:"notifiedAt" format:"date-time" doc:"When the Admin was notified"`
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
	WorkID  string `json:"workId" format:"uuid"`
	Title   string `json:"title"`
	Files   int    `json:"files" minimum:"0"`
	Seconds int64  `json:"seconds" minimum:"0"`
	Bytes   int64  `json:"bytes" minimum:"0"`
	AddedAt string `json:"addedAt" format:"date-time"`
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

func noMade() madeImpact {
	return madeImpact{Nodes: []string{}, Unconvertible: []string{}, Rows: []madeRow{}}
}

func noRemake() remakeImpact { return remakeImpact{Rows: []madeRow{}} }

type backfillRest struct {
	ProfileID string `json:"profileId,omitempty" format:"uuid" doc:"An existing Profile the rest move to"`
	NewName   string `json:"newName,omitempty" maxLength:"100" doc:"The name of a new Profile, the old one without the made-from Target, that the rest move to"`
}

type backfill struct {
	Convert []string     `json:"convert" nullable:"false" format:"uuid" doc:"The Works to convert"`
	Rest    backfillRest `json:"rest" doc:"Where the other Works go"`
	Default string       `json:"default" format:"uuid" doc:"The Profile new Works take"`
}

// conversionRequested refuses a Conversion field the build can't save
// yet, unless it is the field's default.
func conversionRequested(field string, set bool) error {
	if !set {
		return nil
	}
	return notyet.Field.On(field, "SlipStream can't make files from other files yet.")
}
