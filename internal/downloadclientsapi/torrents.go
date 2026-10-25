package downloadclientsapi

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	torrentNotFound = api.NewCode(http.StatusNotFound, "torrent_not_found", "No Torrent has this id")
	noPreview       = api.NewCode(http.StatusBadRequest, "no_preview", "Only remove, mark failed and recheck can be previewed")
)

// TorrentRow is one Torrent as the Torrents view lists it.
type TorrentRow struct {
	ID          string       `json:"id" format:"uuid"`
	NodeID      string       `json:"nodeId" format:"uuid"`
	Name        string       `json:"name" doc:"The release's name in its metainfo, shown for a Torrent whose Work was deleted"`
	ContentType string       `json:"contentType" doc:"The Content Type it was first grabbed for"`
	WorkID      *string      `json:"workId,omitempty" format:"uuid" doc:"Its newest Grab's Work, absent once no Grab is left"`
	Edition     string       `json:"edition,omitempty" doc:"Its newest Grab's Edition"`
	Quality     string       `json:"quality,omitempty" enum:"sdtv,dvd,webrip-480p,hdtv-720p,webrip-720p,webdl-720p,bluray-720p,hdtv-1080p,webrip-1080p,webdl-1080p,bluray-1080p,remux-1080p,hdtv-2160p,webrip-2160p,webdl-2160p,bluray-2160p,remux-2160p,cam" doc:"The quality its newest Grab's release title names"`
	State       string       `json:"state" enum:"removing,failed,tracker-error,paused,checking,queued,stalled,downloading,seeding"`
	Hold        string       `json:"hold,omitempty" enum:"wanted,window,disk_full,slot,check_slot,disk_guard,draining,controller" doc:"Why its Node holds it back, as its Node last reported"`
	Until       *time.Time   `json:"until,omitempty" doc:"When a window hold starts it, or a check resumes it"`
	InLine      int          `json:"queuePosition,omitempty" minimum:"0" doc:"Its place, from 1, among its Node's downloads waiting for a slot"`
	Severity    string       `json:"severity" enum:"none,warning,failure"`
	Tabs        []string     `json:"tabs" enum:"downloading,seeding,problems,all" nullable:"false" doc:"The Torrents view's tabs it shows under"`
	Pending     string       `json:"pending,omitempty" enum:"pause,resume" doc:"An Admin's pause or resume its Node hasn't applied yet"`
	Size        int64        `json:"size" minimum:"0" doc:"Bytes wanted"`
	Done        int64        `json:"done" minimum:"0" doc:"Bytes done as its Node last reported them; torrents.live carries the live figure"`
	Uploaded    int64        `json:"uploaded" minimum:"0" doc:"All-time bytes uploaded"`
	Ratio       float64      `json:"ratio" minimum:"0" doc:"Uploaded over the larger of downloaded and size"`
	Seeding     int64        `json:"seedingSeconds" minimum:"0" doc:"All-time seconds seeded"`
	Tracker     *trackerCell `json:"tracker,omitempty" doc:"Absent when its metainfo names no tracker"`
	Error       string       `json:"error,omitempty" doc:"The engine's error message for the failure"`
	Fault       *fault       `json:"fault,omitempty" doc:"The fault found in the release itself, once there is one"`
	LastKnown   *time.Time   `json:"lastKnown,omitempty" doc:"When its Node last reported it, set while the Node is out of step"`
	AddedAt     time.Time    `json:"addedAt"`
}

type trackerCell struct {
	Host    string     `json:"host" doc:"Empty when its tracker list is unknown"`
	Status  string     `json:"status" enum:"working,updating,error,unregistered,not_contacted"`
	Message string     `json:"message,omitempty" doc:"The tracker's failure reason or warning, verbatim"`
	RetryAt *time.Time `json:"retryAt,omitempty" doc:"When it tries a failing tracker again"`
}

type fault struct {
	Kind     string    `json:"kind" enum:"engine_error,unregistered,no_payload"`
	At       time.Time `json:"at"`
	Sentence string    `json:"sentence" doc:"The fault, as a sentence"`
}

// faultWords is why each fault makes a release look bad.
var faultWords = map[downloadclients.FaultKind]string{
	downloadclients.FaultEngineError:  "The torrent client found an error in the release's data.",
	downloadclients.FaultUnregistered: "The release's tracker reports that the torrent is no longer registered.",
	downloadclients.FaultNoPayload:    "No data arrived at any point while it was downloading.",
}

var (
	tabWords = map[downloadclients.Tab]string{
		downloadclients.TabDownloading: "downloading", downloadclients.TabSeeding: "seeding",
		downloadclients.TabProblems: "problems", downloadclients.TabAll: "all",
	}
	severityWords = map[downloadclients.Severity]string{
		downloadclients.SeverityNone: "none", downloadclients.SeverityWarning: "warning", downloadclients.SeverityFailure: "failure",
	}
	pendingWords = map[downloadclients.Wanted]string{downloadclients.Paused: "pause", downloadclients.Started: "resume"}
)

func rowOf(r downloadclients.Row) TorrentRow {
	out := TorrentRow{
		ID: r.ID.String(), NodeID: r.Node.String(), Name: r.Name, ContentType: string(r.ContentType),
		State: string(r.State), Hold: string(r.Hold), Until: when(r.Until), InLine: r.QueuePosition,
		Severity: severityWords[r.Severity], Tabs: []string{}, Pending: pendingWords[r.Pending],
		Size: r.Size, Done: r.Done, Uploaded: r.Totals.Uploaded, Ratio: downloadclients.Ratio(r.Totals, r.Size),
		Seeding: int64(r.Totals.Seeding / time.Second), Error: r.Error, LastKnown: when(r.LastKnown), AddedAt: r.AddedAt,
	}
	for _, t := range downloadclients.Tabs {
		if r.Tabs&t != 0 {
			out.Tabs = append(out.Tabs, tabWords[t])
		}
	}
	if len(r.Uses) > 0 {
		newest := r.Uses[0]
		out.WorkID, out.Edition = new(newest.Work.String()), newest.Edition
		if q, ok := release.Title(newest.Release, nil).Quality.Get(); ok {
			out.Quality = q.String()
		}
	}
	if c := r.Tracker; c != nil {
		out.Tracker = &trackerCell{Host: c.Host, Status: string(c.Status), Message: c.Message, RetryAt: when(c.RetryAt)}
	}
	if f := r.Fault; f != nil {
		out.Fault = &fault{Kind: string(f.Kind), At: f.At, Sentence: faultWords[f.Kind]}
	}
	return out
}

// when is t, nil for the zero time.
func when(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

type listInput struct {
	IDs         []string `query:"ids" format:"uuid" doc:"Only these Torrents, such as the ones a torrents.changed named"`
	Tab         string   `query:"tab" enum:"downloading,seeding,problems,all"`
	NodeID      string   `query:"nodeId" format:"uuid"`
	ContentType string   `query:"contentType"`
	Q           string   `query:"q" maxLength:"200" doc:"Matches the release name, the Grabs' release titles and their Works' labels, ignoring case"`
}

type torrentList struct {
	Rows   []TorrentRow `json:"rows" nullable:"false" doc:"In the order they were added"`
	Counts counts       `json:"counts" doc:"Every Torrent live or Removing, whatever the filter"`
}

type counts struct {
	ByState []stateCount `json:"byState" nullable:"false" doc:"Each state, in order of precedence"`
	ByTab   []tabCount   `json:"byTab" nullable:"false" doc:"Each tab, in the view's order"`
	ByNode  []nodeCount  `json:"byNode" nullable:"false" doc:"Each Node holding any, in Node order"`
}

type stateCount struct {
	State    string `json:"state" enum:"removing,failed,tracker-error,paused,checking,queued,stalled,downloading,seeding"`
	Torrents int    `json:"torrents" minimum:"0"`
}

type tabCount struct {
	Tab      string `json:"tab" enum:"downloading,seeding,problems,all"`
	Torrents int    `json:"torrents" minimum:"0"`
}

type nodeCount struct {
	NodeID   string `json:"nodeId" format:"uuid"`
	Torrents int    `json:"torrents" minimum:"0"`
}

type torrentListOutput struct{ Body torrentList }

func filterOf(in *listInput) (downloadclients.Filter, error) {
	f := downloadclients.Filter{Type: works.TypeID(in.ContentType), Search: in.Q}
	for t, w := range tabWords {
		if w == in.Tab {
			f.Tab = t
		}
	}
	if in.NodeID != "" {
		u, err := storage.ParseUUID(in.NodeID)
		if err != nil {
			return f, err
		}
		f.Node = role.NodeID{UUID: u}
	}
	if in.IDs != nil {
		f.IDs = []downloadclients.ID{}
	}
	for _, s := range in.IDs {
		id, err := torrentID(s)
		if err != nil {
			return f, err
		}
		f.IDs = append(f.IDs, id)
	}
	return f, nil
}

func torrentID(s string) (downloadclients.ID, error) {
	u, err := storage.ParseUUID(s)
	return downloadclients.ID{UUID: u}, err
}

func countsOf(c downloadclients.Counts) counts {
	out := counts{ByState: []stateCount{}, ByTab: []tabCount{}, ByNode: []nodeCount{}}
	for _, s := range downloadclients.States {
		out.ByState = append(out.ByState, stateCount{State: string(s), Torrents: c.States[s]})
	}
	for _, t := range downloadclients.Tabs {
		out.ByTab = append(out.ByTab, tabCount{Tab: tabWords[t], Torrents: c.Tabs[t]})
	}
	for _, node := range slices.SortedFunc(maps.Keys(c.Nodes), func(a, b role.NodeID) int { return a.Compare(b.UUID) }) {
		out.ByNode = append(out.ByNode, nodeCount{NodeID: node.String(), Torrents: c.Nodes[node]})
	}
	return out
}

func (h handlers) listTorrents(ctx context.Context, _ identity.Admin, in *listInput) (*torrentListOutput, error) {
	clients, _ := open(h.clients)
	f, err := filterOf(in)
	if err != nil {
		return nil, err
	}
	l, err := clients.List(ctx, f)
	if err != nil {
		return nil, err
	}
	out := torrentList{Rows: make([]TorrentRow, len(l.Rows)), Counts: countsOf(l.Counts)}
	for i, r := range l.Rows {
		out.Rows[i] = rowOf(r)
	}
	return &torrentListOutput{Body: out}, nil
}

// TorrentDetail is one Torrent as the Inspector shows it.
type TorrentDetail struct {
	TorrentRow
	Downloaded int64           `json:"downloaded" minimum:"0" doc:"All-time bytes downloaded"`
	Active     int64           `json:"activeSeconds" minimum:"0" doc:"All-time seconds it ran"`
	InfoHash   string          `json:"infoHash" doc:"The v1 info hash in hex"`
	SaveFolder string          `json:"saveFolder,omitempty" doc:"Where its Node keeps it, absent until its Node reports it"`
	PieceSize  int64           `json:"pieceSize" minimum:"1" doc:"Bytes"`
	Private    bool            `json:"private"`
	Trackers   []trackerDetail `json:"trackers" nullable:"false" doc:"In its metainfo's order"`
	Files      []fileDetail    `json:"files" nullable:"false" doc:"In its metainfo's order"`
	Grabs      []torrentGrab   `json:"grabs" nullable:"false" doc:"The Grabs that used it, newest first"`
	Live       bool            `json:"live" doc:"Its Node responded, so trackers and files are live. Otherwise they are from its last report"`
	AnnounceAt *time.Time      `json:"announceAt,omitempty" doc:"The earliest every tracker accepts an announce, set while live"`
}

type trackerDetail struct {
	URL          string     `json:"url" doc:"With its passkey hidden"`
	Tier         int        `json:"tier" minimum:"0"`
	Status       string     `json:"status" enum:"working,updating,error,unregistered,not_contacted"`
	Message      string     `json:"message,omitempty" doc:"The tracker's failure reason or warning, verbatim"`
	RetryAt      *time.Time `json:"retryAt,omitempty" doc:"When it tries a failing tracker again, from the last report"`
	AnsweredAt   *time.Time `json:"answeredAt,omitempty" doc:"When it last responded, set while live"`
	NextAnnounce *time.Time `json:"nextAnnounce,omitempty" doc:"When its next announce is due, set while live"`
	MinInterval  int64      `json:"minIntervalSeconds,omitempty" minimum:"0" doc:"Its min interval as the announce policy holds to it, set while live"`
	AcceptsAt    *time.Time `json:"acceptsAt,omitempty" doc:"The earliest it accepts an announce, set while live"`
}

type fileDetail struct {
	Index  int     `json:"index" minimum:"0"`
	Path   string  `json:"path" doc:"Under the save folder, with / between folders"`
	Size   int64   `json:"size" minimum:"0"`
	Done   *int64  `json:"done,omitempty" minimum:"0" doc:"Bytes done, absent while unknown"`
	ItemID *string `json:"itemId,omitempty" format:"uuid" doc:"The Item it filled once a Grab imported it"`
}

type torrentGrab struct {
	ID      string `json:"id" format:"uuid"`
	WorkID  string `json:"workId" format:"uuid"`
	ItemID  string `json:"itemId" format:"uuid"`
	Release string `json:"release"`
	Stage   string `json:"stage" enum:"downloading,importing,imported,failed"`
}

func detailOf(d downloadclients.Detail) TorrentDetail {
	out := TorrentDetail{
		TorrentRow: rowOf(d.Row), Downloaded: d.Totals.Downloaded, Active: int64(d.Totals.Active / time.Second),
		InfoHash: d.InfoHash.String(), SaveFolder: d.SaveFolder, PieceSize: d.PieceSize, Private: d.Private,
		Trackers: []trackerDetail{}, Files: []fileDetail{}, Grabs: []torrentGrab{}, Live: d.Live, AnnounceAt: when(d.AnnounceAt),
	}
	for _, t := range d.Trackers {
		out.Trackers = append(out.Trackers, trackerDetail{
			URL: t.URL, Tier: t.Tier, Status: string(t.Status), Message: t.Message, RetryAt: when(t.RetryAt),
			AnsweredAt: when(t.AnsweredAt), NextAnnounce: when(t.NextAnnounce), MinInterval: int64(t.MinInterval / time.Second), AcceptsAt: when(t.AcceptsAt),
		})
	}
	for _, f := range d.Files {
		fd := fileDetail{Index: f.Index, Path: f.Path, Size: f.Size, Done: f.Done}
		if !f.Item.IsZero() {
			fd.ItemID = new(f.Item.String())
		}
		out.Files = append(out.Files, fd)
	}
	for _, u := range d.Uses {
		out.Grabs = append(out.Grabs, torrentGrab{ID: u.Grab.String(), WorkID: u.Work.String(), ItemID: u.Item.String(), Release: u.Release, Stage: string(u.Stage)})
	}
	return out
}

type torrentInput struct {
	ID string `path:"id" format:"uuid"`
}

type torrentOutput struct{ Body TorrentDetail }

func (h handlers) getTorrent(ctx context.Context, _ identity.Admin, in *torrentInput) (*torrentOutput, error) {
	clients, _ := open(h.clients)
	id, err := torrentID(in.ID)
	if err != nil {
		return nil, torrentNotFound
	}
	d, err := clients.Get(ctx, id)
	if errors.Is(err, downloadclients.ErrNotFound) {
		return nil, torrentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &torrentOutput{Body: detailOf(d)}, nil
}

type actInput struct {
	Body struct {
		Action      string   `json:"action" enum:"pause,resume,announce,recheck,mark-failed,remove"`
		IDs         []string `json:"ids" format:"uuid" minItems:"1" nullable:"false"`
		DeleteFiles bool     `json:"deleteFiles,omitempty" doc:"remove: also delete each download copy"`
		DryRun      bool     `json:"dryRun,omitempty" doc:"remove, mark-failed and recheck: return what it would do and change nothing"`
	}
}

type acted struct {
	Results []actResult `json:"results" nullable:"false" doc:"One for each id, in the order sent"`
	Preview *preview    `json:"preview,omitempty" doc:"What a dry run would do"`
}

type actResult struct {
	ID      string   `json:"id" format:"uuid"`
	Done    bool     `json:"done"`
	Refusal *refusal `json:"refusal,omitempty" doc:"Why it wasn't done"`
}

type refusal struct {
	Code      string     `json:"code" enum:"not_found,removed,importing,not_stalled,offline,window,not_running,failed"`
	WorkID    *string    `json:"workId,omitempty" format:"uuid" doc:"importing: the Work whose import holds it"`
	AcceptsAt *time.Time `json:"acceptsAt,omitempty" doc:"window: when every tracker accepts an announce"`
}

type preview struct {
	Cancels     []canceled `json:"cancels" nullable:"false" doc:"The Grabs at downloading that a remove fails"`
	StopSeeding []string   `json:"stopSeeding" format:"uuid" nullable:"false" doc:"The Works whose complete Torrents stop seeding"`
	Frees       *int64     `json:"frees" minimum:"0" doc:"Bytes deleting the download copies frees, null while a Node can't report it"`
	Linked      bool       `json:"linked" doc:"Some file is also a library file, as a recheck reads it"`
}

type canceled struct {
	GrabID  string `json:"grabId" format:"uuid"`
	WorkID  string `json:"workId" format:"uuid"`
	Release string `json:"release"`
}

type actOutput struct{ Body acted }

func (h handlers) actOnTorrents(ctx context.Context, _ identity.Admin, in *actInput) (*actOutput, error) {
	clients, _ := open(h.clients)
	a := downloadclients.Action{Kind: downloadclients.ActionKind(in.Body.Action), DeleteFiles: in.Body.DeleteFiles, DryRun: in.Body.DryRun}
	for _, s := range in.Body.IDs {
		id, err := torrentID(s)
		if err != nil {
			return nil, err
		}
		a.IDs = append(a.IDs, id)
	}
	results, p, err := clients.Act(ctx, a)
	if errors.Is(err, downloadclients.ErrNoPreview) {
		return nil, noPreview
	}
	if err != nil {
		return nil, err
	}
	out := acted{Results: make([]actResult, len(results)), Preview: previewOf(p)}
	for i, r := range results {
		out.Results[i] = actResult{ID: r.ID.String(), Done: r.Refusal == nil}
		if f := r.Refusal; f != nil {
			out.Results[i].Refusal = &refusal{Code: string(f.Code), AcceptsAt: when(f.AcceptsAt)}
			if !f.Work.IsZero() {
				out.Results[i].Refusal.WorkID = new(f.Work.String())
			}
		}
	}
	return &actOutput{Body: out}, nil
}

func previewOf(p *downloadclients.Preview) *preview {
	if p == nil {
		return nil
	}
	out := &preview{Cancels: []canceled{}, StopSeeding: []string{}, Frees: p.Frees, Linked: p.Linked}
	for _, u := range p.Cancels {
		out.Cancels = append(out.Cancels, canceled{GrabID: u.Grab.String(), WorkID: u.Work.String(), Release: u.Release})
	}
	for _, w := range p.StopSeeding {
		out.StopSeeding = append(out.StopSeeding, w.String())
	}
	return out
}
