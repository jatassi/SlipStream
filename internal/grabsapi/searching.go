package grabsapi

import (
	"context"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
)

// cellKey is a cell as the wire names it.
type cellKey struct {
	ItemID   string `json:"itemId" format:"uuid"`
	Edition  string `json:"edition" doc:"any under any cut, else the key of the cut the cell wants"`
	TargetID string `json:"targetId" format:"uuid"`
}

type workSearching struct {
	Groups []groupState `json:"groups" nullable:"false" doc:"Each search group of the Work, in tree order"`
	Waits  []waiting    `json:"waits" nullable:"false" doc:"Each cell waiting for a better Release, by when it grabs"`
	Stops  []stopped    `json:"stops" nullable:"false" doc:"Each cell SlipStream stopped searching again for after failed Grabs"`
	Lead   lead         `json:"lead" doc:"The one line the Work's header shows"`
}

type groupState struct {
	Key       string               `json:"key" doc:"The search group's key, to compare and never to parse"`
	Label     string               `json:"label" doc:"The group in words, such as season 3"`
	State     string               `json:"state" enum:"idle,queued,searching"`
	Lane      *string              `json:"lane" enum:"now,bulk,backlog" doc:"The queued entry's lane, else the running search's; null while idle"`
	Trigger   *string              `json:"trigger" enum:"search,search_all,add,monitor,profile,editions,retarget,release_date,refresh,backlog,failed_grab" doc:"What queued it, else what started the running search; null while idle"`
	Again     bool                 `json:"again" doc:"Queued again behind its own running search"`
	Behind    api.Nullable[behind] `json:"behind" doc:"What it waits on; null when nothing holds it"`
	LastAt    *time.Time           `json:"lastSearchedAt" doc:"When its last search finished; null before one did"`
	Last      *string              `json:"lastResult" enum:"grabbed,waiting,nothing_found,nothing_acceptable,no_indexer,no_root" doc:"How its last search finished; null before one did"`
	LastFound int                  `json:"lastFound" minimum:"0" doc:"The Releases its last search found"`
	LastGrabs int                  `json:"lastGrabs" minimum:"0" doc:"The Grabs its last search made or joined"`
	NextAt    *time.Time           `json:"nextSearchAt" doc:"When Backlog search searches it next; null while queued or searching, and while Backlog search is off"`
}

type behind struct {
	Ahead       int        `json:"ahead" minimum:"0" doc:"Works ahead of it in its lane"`
	AheadLane   *string    `json:"aheadLane" enum:"now,bulk,backlog" doc:"The lane of the Works ahead; null when none are"`
	AheadFrom   *string    `json:"aheadFrom" enum:"search,search_all,add,monitor,profile,editions,retarget,release_date,refresh,backlog,failed_grab" doc:"What queued the Works ahead, as in Queued behind 140 Works from Search all; null when none are"`
	IndexerID   *string    `json:"indexerId" format:"uuid" doc:"The Indexer whose search limit it waits for; null when it waits for none"`
	IndexerName *string    `json:"indexerName" doc:"That Indexer's name; null when it waits for none"`
	Until       *time.Time `json:"until" doc:"When the Indexer's search limit frees a search; null when it waits for none"`
	Starved     bool       `json:"starved" doc:"RSS sync uses all of the Indexer's search limit, so Backlog search waits for it"`
}

type waiting struct {
	Cell       cellKey   `json:"cell"`
	Until      time.Time `json:"until" doc:"When the wait ends and SlipStream grabs the best Release it holds"`
	Candidates int       `json:"candidates" minimum:"1" doc:"The Releases it holds"`
}

type stopped struct {
	Cell     cellKey     `json:"cell"`
	Failures []failureAt `json:"failures" nullable:"false" doc:"The failed Grabs that stopped it, oldest first"`
	NextAt   *time.Time  `json:"nextSearchAt" doc:"When Backlog search searches it next; null while Backlog search is off"`
}

type failureAt struct {
	Code   string    `json:"code" enum:"indexer_unavailable,indexer_removed,indexer_stopped,home_offline,torrent_unreachable,magnet_invalid,no_metadata,no_peer_source,torrent_invalid,infohash_mismatch,torrent_v2_only,download_failed,marked_failed,removed,client_removed,client_deleted,goal_met,no_video_file,probe_failed,fills_nothing,name_taken,name_too_long,transfer_failed,import_failed,node_written_off,root_changed,canceled,blocklisted,release_fault,client_error,declined,unreadable"`
	At     time.Time `json:"at"`
	GrabID *string   `json:"grabId" format:"uuid" doc:"The failed Grab; null once it is pruned"`
}

type lead struct {
	GroupKey *string               `json:"groupKey" doc:"The group the line is about; null for a waiting or idle Work"`
	Of       int                   `json:"of" minimum:"0" doc:"The Work's groups queued or searching, as in Searching season 3 of 10"`
	Wait     api.Nullable[waiting] `json:"wait" doc:"The wait the line is about; null unless the Work only waits"`
}

type workSearchingOutput struct{ Body workSearching }

func searchingRoutes(o *api.Owner, h handlers) {
	api.Register(o, api.Admin, api.Op{
		ID:      "getWorkSearching",
		Method:  http.MethodGet,
		Path:    "/works/{workId}/searching",
		Summary: "Get what Automatic search is doing for a Work: each search group's state, its waits and its stopped cells",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound, notYet},
	}, h.getWorkSearching)
}

func (handlers) getWorkSearching(context.Context, identity.Admin, *workInput) (*workSearchingOutput, error) {
	return nil, notYet
}
