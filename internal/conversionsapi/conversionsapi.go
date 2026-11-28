// Package conversionsapi is S44 (Conversion)'s operations: the
// Conversions tab and its actions, and a Node's Converting section. Every
// operation translates HTTP; conversions decides.
package conversionsapi

import (
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/conversionsapi/notyet"
	"github.com/jatassi/slipstream/internal/setting"
)

var (
	conversionNotFound   = api.NewCode(http.StatusNotFound, "conversion_not_found", "No Conversion has that id")
	conversionFinished   = api.NewCode(http.StatusConflict, "conversion_finished", "This Conversion has finished")
	conversionNotWaiting = api.NewCode(http.StatusConflict, "conversion_not_waiting", "This Conversion isn't waiting to start")
	conversionNotFailed  = api.NewCode(http.StatusConflict, "conversion_not_failed", "This Conversion hasn't failed")
	conversionPlacing    = api.NewCode(http.StatusConflict, "conversion_placing", "This Conversion's file is being placed in the library")
	nodeNotFound         = api.NewCode(http.StatusNotFound, "node_not_found", "No Node has this id")
)

type counts struct {
	Converting int        `json:"converting" minimum:"0"`
	Paused     int        `json:"paused" minimum:"0"`
	Waiting    int        `json:"waiting" minimum:"0"`
	Failed     int        `json:"failed" minimum:"0"`
	ClearsAt   *time.Time `json:"clearsAt" doc:"When the last unfinished Conversion is forecast to finish; null when none is forecast"`
}

type named struct {
	ID   string `json:"id" format:"uuid"`
	Name string `json:"name"`
}

type work struct {
	ID    string `json:"id" format:"uuid"`
	Title string `json:"title"`
	Type  string `json:"type" doc:"The Work's Content Type"`
}

type item struct {
	ID    string `json:"id" format:"uuid"`
	Label string `json:"label" doc:"Such as S01E04"`
}

type target struct {
	ID   *string `json:"id" format:"uuid" doc:"null once the Target is deleted"`
	Name string  `json:"name"`
}

type source struct {
	FileID *string `json:"fileId" format:"uuid" doc:"null once the source file is gone"`
	Name   string  `json:"name" doc:"The source file's name"`
	Target string  `json:"target" doc:"The name of the Target the source file fills"`
}

type progress struct {
	Percent     float64 `json:"percent" minimum:"0" maximum:"100"`
	Speed       float64 `json:"speed" minimum:"0" doc:"Times real time"`
	SecondsLeft int64   `json:"secondsLeft" minimum:"0"`
}

type wait struct {
	Code         string     `json:"code" enum:"in_line,hours,room,copying,away,download,testing,transcode,no_node,no_source" doc:"Why it hasn't started"`
	Node         *named     `json:"node,omitempty" doc:"The Node its words name, when one"`
	Until        *time.Time `json:"until,omitempty" doc:"For hours and download: when the wait ends"`
	Servers      []string   `json:"servers,omitempty" doc:"For transcode: the Media Servers transcoding on its Node"`
	Free         *int64     `json:"free,omitempty" minimum:"0" doc:"For room: the bytes free on the Node"`
	Need         *int64     `json:"need,omitempty" minimum:"0" doc:"For room: the bytes it needs"`
	Place        *int       `json:"place,omitempty" minimum:"1" doc:"For in_line: its place in line, from 1"`
	SourceTarget string     `json:"sourceTarget,omitempty" doc:"For download: the Target it is made from"`
	Cause        string     `json:"cause,omitempty" enum:"no_role,lapsed,runs_elsewhere,pinned_elsewhere,failed_here,untested,test_failed,retesting,proof_lost,processor_off,none_online,no_holder" doc:"For no_node: why the Node closest to running it can't"`
}

type failure struct {
	Code   string   `json:"code" doc:"The fault, such as encoder_failed"`
	Class  string   `json:"class" enum:"run,source" doc:"run: it failed on every Node it may try; source: its source file can't be converted as its settings ask"`
	Nodes  []string `json:"nodes" nullable:"false" doc:"The Nodes its counted attempts failed on"`
	Detail string   `json:"detail" doc:"ffmpeg's last lines, for Details"`
}

type attempt struct {
	N         int        `json:"n" minimum:"1"`
	Node      string     `json:"node" doc:"The Node's name"`
	Outcome   string     `json:"outcome,omitempty" enum:"failed,stalled,disk_full,put_back" doc:"Absent while the attempt runs"`
	Code      string     `json:"code" doc:"The fault, empty when none"`
	StartedAt time.Time  `json:"startedAt"`
	EndedAt   *time.Time `json:"endedAt" doc:"null while the attempt runs"`
}

// ConversionRow is exported so the document carries the fields a row of
// every band embeds.
type ConversionRow struct {
	Kind     string                 `json:"kind" enum:"conversion"`
	ID       string                 `json:"id" format:"uuid"`
	Stage    string                 `json:"stage" enum:"queued,copying,ready,running,paused,checking,returning,landing,done,failed,canceled"`
	Lane     string                 `json:"lane" enum:"moved_up,new,upgrade,backfill"`
	Work     work                   `json:"work"`
	Items    []item                 `json:"items" nullable:"false" doc:"In tree order"`
	Target   target                 `json:"target" doc:"The Target it makes a file for"`
	Source   source                 `json:"source"`
	Node     api.Nullable[named]    `json:"node" doc:"The Node it is placed on; null until it is placed"`
	Progress api.Nullable[progress] `json:"progress" doc:"While it runs; conversions.live keeps it current. null otherwise"`
	PausedBy []string               `json:"pausedBy" doc:"The Media Servers whose transcodes pause it; null while it isn't paused"`
	Wait     api.Nullable[wait]     `json:"wait" doc:"Why it hasn't started; null once it has"`
	Start    *time.Time             `json:"start" doc:"When it is forecast to start; null once it has, or with no forecast"`
	Finish   *time.Time             `json:"finish" doc:"When it is forecast to finish; null with no forecast"`
	Failure  api.Nullable[failure]  `json:"failure" doc:"Why it failed; null unless it failed"`
	Attempts []attempt              `json:"attempts" nullable:"false" doc:"Oldest first"`
}

type conversionGroup struct {
	Kind      string     `json:"kind" enum:"group"`
	ID        string     `json:"id"`
	GroupKind string     `json:"groupKind" enum:"backfill,season"`
	Label     string     `json:"label" doc:"Such as Backfill of 720p mobile, or The Office season 2"`
	Count     int        `json:"count" minimum:"1"`
	Nodes     []string   `json:"nodes" nullable:"false" doc:"The Nodes they are forecast to run on"`
	Seconds   int64      `json:"seconds" minimum:"0" doc:"Their forecast run time"`
	Bytes     int64      `json:"bytes" minimum:"0" doc:"Their forecast output size"`
	Start     *time.Time `json:"start" doc:"null with no forecast"`
	Finish    *time.Time `json:"finish" doc:"null with no forecast"`
}

type upNext struct{}

func (upNext) Variants() (discriminator string, variants []any) {
	return "kind", []any{ConversionRow{}, conversionGroup{}}
}

type laneRows struct {
	Lane string              `json:"lane" enum:"moved_up,new,upgrade,backfill"`
	Rows []api.OneOf[upNext] `json:"rows" nullable:"false" doc:"In run order"`
}

type conversionList struct {
	Counts     counts          `json:"counts"`
	Attention  []ConversionRow `json:"attention" nullable:"false" doc:"Failed Conversions, newest first"`
	Converting []ConversionRow `json:"converting" nullable:"false" doc:"Running and paused Conversions"`
	Next       []laneRows      `json:"next" nullable:"false" doc:"Up next, one entry per lane in run order"`
	Cursor     *string         `json:"cursor" doc:"Pages Up next; null on its last page"`
}

type listInput struct {
	Band   string `query:"band" enum:"attention,converting,next" doc:"Only this band; left out, every band"`
	Node   string `query:"node" format:"uuid" doc:"Only Conversions placed or forecast on this Node"`
	Type   string `query:"type" doc:"Only Works of this Content Type"`
	Q      string `query:"q" maxLength:"200" doc:"Only Works whose title holds this"`
	Work   string `query:"work" format:"uuid" doc:"Only this Work's Conversions"`
	Cursor string `query:"cursor" doc:"The cursor of the previous page"`
}

type listOutput struct{ Body conversionList }

type groupInput struct {
	GroupID string `path:"groupId"`
	Cursor  string `query:"cursor" doc:"The cursor of the previous page"`
}

type groupPage struct {
	Rows   []ConversionRow `json:"rows" nullable:"false" doc:"In run order"`
	Cursor *string         `json:"cursor" doc:"null on the last page"`
}

type groupOutput struct{ Body groupPage }

type conversionInput struct {
	ID string `path:"id" format:"uuid"`
}

type moveInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		After api.Patch[string] `json:"after" doc:"The Conversion it goes right after, in its lane; null for the head of Moved up"`
	}
}

type rowOutput struct{ Body ConversionRow }

type groupCancelInput struct {
	GroupID string `path:"groupId"`
}

type canceledOutput struct {
	Body struct {
		Canceled int `json:"canceled" minimum:"0"`
	}
}

type hours struct {
	On     bool          `json:"on" doc:"Whether the Node converts only inside the window"`
	Window setting.Times `json:"window"`
}

type sentence struct {
	API       *string `json:"api" enum:"qsv,vaapi,cuda,videotoolbox" doc:"The hardware it converts with; null with none"`
	Processor bool    `json:"processor" doc:"Whether it converts with its processor"`
	None      bool    `json:"none" doc:"Whether it can convert nothing"`
}

type shape struct {
	Codec  string `json:"codec" enum:"h264,hevc,av1"`
	Depth  int    `json:"depth" enum:"8,10"`
	Range  string `json:"range" enum:"sdr,hdr10,source"`
	KeepDV bool   `json:"keepDolbyVision"`
	Curve  string `json:"curve" enum:"bt2390,hable,mobius,reinhard"`
}

type proof struct {
	Shape    shape      `json:"shape"`
	Class    string     `json:"class" enum:"h264_8,hevc_8,hevc_10,av1_8,av1_10,hdr10,hlg,dv5,dv81,decode_sdr,decode_hdr10"`
	Path     string     `json:"path" enum:"hardware,hardware_opencl,processor"`
	API      *string    `json:"api" enum:"qsv,vaapi,cuda,videotoolbox" doc:"null for the processor"`
	Device   string     `json:"device" doc:"Such as Intel UHD Graphics 770"`
	Speed    float64    `json:"speed" minimum:"0" doc:"Times real time at 1080p"`
	Pauses   bool       `json:"pauses" doc:"Whether a pause stops its progress"`
	ProvenAt *time.Time `json:"provenAt" doc:"null unless it passed"`
	FailedAt *time.Time `json:"failedAt" doc:"null unless it failed"`
	LostAt   *time.Time `json:"lostAt" doc:"When a driver change lost it; null otherwise"`
	Error    string     `json:"error" doc:"ffmpeg's last lines, for Details; empty unless it failed"`
}

type volume struct {
	Free  int64 `json:"free" minimum:"0"`
	Total int64 `json:"total" minimum:"0"`
}

type nodeConverting struct {
	AtOnce        int                  `json:"atOnce" minimum:"1"`
	AtOnceDefault int                  `json:"atOnceDefault" minimum:"1"`
	Hours         hours                `json:"hours"`
	Processor     bool                 `json:"processor" doc:"Use the processor when no GPU can"`
	Sentence      sentence             `json:"sentence"`
	Proofs        []proof              `json:"proofs" nullable:"false"`
	Scratch       api.Nullable[volume] `json:"scratch" doc:"The scratch folder's volume; null while unknown"`
	Override      bool                 `json:"override" doc:"Whether a Target's Run on names this Node while it can't run that Target's Conversions"`
	Running       int                  `json:"running" minimum:"0"`
}

type nodeInput struct {
	ID string `path:"id" format:"uuid"`
}

type nodeSettings struct {
	AtOnce    *int  `json:"atOnce" minimum:"1" maximum:"16" doc:"null for the default"`
	Hours     hours `json:"hours"`
	Processor bool  `json:"processor"`
}

type updateNodeInput struct {
	ID   string `path:"id" format:"uuid"`
	Body nodeSettings
}

type nodeOutput struct{ Body nodeConverting }

type handlers struct{}

func Routes(r *api.Router) {
	o := r.Owner("conversions", api.Admin)
	h := handlers{}

	api.Register(o, api.Admin, api.Op{
		ID: "listConversions", Method: http.MethodGet, Path: "/conversions",
		Summary: "List the Conversions tab: its counts, the ones that need attention, the ones converting, and a page of Up next by lane",
		Status:  http.StatusOK, Errors: []*api.Code{notyet.Op}, Gzip: true,
	}, h.listConversions)

	api.Register(o, api.Admin, api.Op{
		ID: "getConversionGroup", Method: http.MethodGet, Path: "/conversions/groups/{groupId}",
		Summary: "Get a page of a folded group's Conversions",
		Status:  http.StatusOK, Errors: []*api.Code{conversionNotFound, notyet.Op},
	}, h.getConversionGroup)

	api.Register(o, api.Admin, api.Op{
		ID: "moveConversionUp", Method: http.MethodPost, Path: "/conversions/{id}/move-up",
		Summary: "Put a waiting Conversion at the head of Moved up",
		Status:  http.StatusOK, Errors: []*api.Code{conversionNotFound, conversionNotWaiting, notyet.Op},
	}, h.moveConversionUp)

	api.Register(o, api.Admin, api.Op{
		ID: "moveConversion", Method: http.MethodPost, Path: "/conversions/{id}/move",
		Summary: "Put a waiting Conversion right after another in its lane",
		Status:  http.StatusOK, Errors: []*api.Code{conversionNotFound, conversionNotWaiting, notyet.Op},
	}, h.moveConversion)

	api.Register(o, api.Admin, api.Op{
		ID: "retryConversion", Method: http.MethodPost, Path: "/conversions/{id}/retry",
		Summary: "Queue a failed Conversion again at the head of its lane, with a fresh set of attempts",
		Status:  http.StatusOK, Errors: []*api.Code{conversionNotFound, conversionNotFailed, notyet.Op},
	}, h.retryConversion)

	api.Register(o, api.Admin, api.Op{
		ID: "cancelConversion", Method: http.MethodPost, Path: "/conversions/{id}/cancel",
		Summary: "Cancel an unfinished Conversion. SlipStream doesn't queue it again until its source file changes or an Admin asks for it",
		Status:  http.StatusOK, Errors: []*api.Code{conversionNotFound, conversionFinished, conversionPlacing, notyet.Op},
	}, h.cancelConversion)

	api.Register(o, api.Admin, api.Op{
		ID: "cancelConversionGroup", Method: http.MethodPost, Path: "/conversions/groups/{groupId}/cancel",
		Summary: "Cancel every unfinished Conversion of a folded group",
		Status:  http.StatusOK, Errors: []*api.Code{conversionNotFound, notyet.Op},
	}, h.cancelConversionGroup)

	api.Register(o, api.Admin, api.Op{
		ID: "getNodeConverting", Method: http.MethodGet, Path: "/nodes/{id}/converting",
		Summary: "Get a Node's Converting section: its settings, what it converts with, and its test runs",
		Status:  http.StatusOK, Errors: []*api.Code{nodeNotFound, notyet.Op},
	}, h.getNodeConverting)

	api.Register(o, api.Admin, api.Op{
		ID: "updateNodeConverting", Method: http.MethodPut, Path: "/nodes/{id}/converting",
		Summary: "Change a Node's Conversions at once, hours and processor switch",
		Status:  http.StatusOK, Errors: []*api.Code{nodeNotFound, notyet.Op},
	}, h.updateNodeConverting)

	api.Register(o, api.Admin, api.Op{
		ID: "rerunNodeTests", Method: http.MethodPost, Path: "/nodes/{id}/converting/rerun",
		Summary: "Run a Node's test runs again",
		Status:  http.StatusAccepted, Errors: []*api.Code{nodeNotFound, notyet.Op},
	}, h.rerunNodeTests)
}
