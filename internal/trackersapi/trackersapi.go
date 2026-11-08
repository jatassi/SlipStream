// Package trackersapi is every /trackers operation: a Tracker's identity,
// which indexers keeps, and its seeding side, which trackers keeps.
package trackersapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers"
)

type Registry interface {
	// Indexers is the Indexers, false unless the Node is a Controller.
	Indexers() (*indexers.Indexers, bool)
	// Trackers is the Trackers' seeding side, false unless the Node is a
	// Controller.
	Trackers() (*trackers.Trackers, bool)
}

type handlers struct{ registry Registry }

func (h handlers) indexers() *indexers.Indexers {
	x, ok := h.registry.Indexers()
	if !ok {
		panic("trackersapi: an operation ran on a Node that is no Controller")
	}
	return x
}

func (h handlers) trackers() *trackers.Trackers {
	x, ok := h.registry.Trackers()
	if !ok {
		panic("trackersapi: an operation ran on a Node that is no Controller")
	}
	return x
}

var (
	trackerNotFound     = api.NewCode(http.StatusNotFound, "tracker_not_found", "That Tracker no longer exists. It may have been merged into another Tracker.")
	trackerNameTaken    = api.NewCode(http.StatusConflict, "tracker_name_taken", "Another Tracker has that name")
	trackerMergeSelf    = api.NewCode(http.StatusUnprocessableEntity, "tracker_merge_self", "A Tracker can't be merged into itself")
	trackerPublic       = api.NewCode(http.StatusConflict, "tracker_public", "Public Trackers have no seeding rule")
	seedingValueRefused = api.NewCode(http.StatusUnprocessableEntity, "seeding_value_refused", "A seeding value is out of range")
)

func trackerAnswer(err error) error {
	switch {
	case errors.Is(err, indexers.ErrNoTracker):
		return trackerNotFound
	case errors.Is(err, indexers.ErrTrackerNameTaken):
		return trackerNameTaken
	case errors.Is(err, indexers.ErrMergeSelf):
		return trackerMergeSelf
	}
	return err
}

// Tracker is a Tracker as every /trackers response shows it.
type Tracker struct {
	ID         string             `json:"id" format:"uuid"`
	Name       string             `json:"name"`
	Privacy    string             `json:"privacy" enum:"private,semi-private,public,unknown"`
	Hosts      []string           `json:"hosts" nullable:"false" doc:"The announce hosts of its private torrents"`
	Keys       []string           `json:"keys" nullable:"false" doc:"How a Release finds it: definition:<id>, prowlarr:<feed Indexer>:<source id> or named:<Indexer>"`
	IndexerIDs []string           `json:"indexerIds" nullable:"false" doc:"The Indexers that reach it, by name"`
	Rule       api.Nullable[rule] `json:"rule" doc:"Its seeding rule in force; null while none is known, and always for a public Tracker"`
	Goal       api.Nullable[goal] `json:"goal" doc:"Its own seeding goal; null while it follows the default goal"`
}

func trackerOf(s trackers.Summary) Tracker {
	out := Tracker{
		ID: s.ID.String(), Name: s.Name, Privacy: string(s.Privacy),
		Hosts: append([]string{}, s.Hosts...), Keys: append([]string{}, s.Keys...), IndexerIDs: []string{},
		Rule: ruleOf(s.Rules),
	}
	for _, id := range s.Indexers {
		out.IndexerIDs = append(out.IndexerIDs, id.String())
	}
	if s.Goal != nil {
		out.Goal = api.Some(goalOf(s.Goal))
	}
	return out
}

type trackerItem struct {
	Tracker
	Counts counts `json:"counts" doc:"Its torrents, live or being removed, as Activity > Torrents counts them"`
}

type counts struct {
	Torrents int `json:"torrents" minimum:"0"`
	Seeding  int `json:"seeding" minimum:"0"`
	Owing    int `json:"owing" minimum:"0" doc:"The ones that still owe it seeding"`
	AtRisk   int `json:"atRisk" minimum:"0" doc:"The ones that owe it seeding and aren't seeding"`
	GoalMet  int `json:"goalMet" minimum:"0" doc:"The ones whose seeding goal is met"`
}

func countsOf(c trackers.Counts) counts {
	return counts{Torrents: c.Torrents, Seeding: c.Seeding, Owing: c.Owing, AtRisk: c.AtRisk, GoalMet: c.GoalMet}
}

type trackerList struct {
	Trackers []trackerItem `json:"trackers" nullable:"false" doc:"By name"`
}

type trackerListOutput struct{ Body trackerList }

func (h handlers) listTrackers(ctx context.Context, by identity.Admin, _ *struct{}) (*trackerListOutput, error) {
	list, err := h.trackers().List(ctx, by)
	if err != nil {
		return nil, err
	}
	out := trackerList{Trackers: []trackerItem{}}
	for _, l := range list {
		out.Trackers = append(out.Trackers, trackerItem{Tracker: trackerOf(l.Summary), Counts: countsOf(l.Counts)})
	}
	return &trackerListOutput{Body: out}, nil
}

type trackerPage struct {
	Tracker
	Stats stats `json:"stats" doc:"SlipStream's own sums over its torrents"`
}

type stats struct {
	Uploaded     int64   `json:"uploaded" minimum:"0" doc:"Bytes, counting each torrent once, live or removed"`
	Downloaded   int64   `json:"downloaded" minimum:"0" doc:"Bytes, counting each torrent once, live or removed"`
	Ratio        float64 `json:"ratio" minimum:"0" doc:"Uploaded over downloaded, or over size where a torrent's size is larger"`
	Torrents     int     `json:"torrents" minimum:"0" doc:"Its torrents, live or being removed"`
	SeedingBytes int64   `json:"seedingBytes" minimum:"0" doc:"The size of its torrents that are seeding"`
	Grabs30d     int     `json:"grabs30d" minimum:"0" doc:"Its Grabs in the last 30 days"`
	Owing        int     `json:"owing" minimum:"0" doc:"Its torrents that still owe it seeding"`
}

type trackerPageOutput struct{ Body trackerPage }

type trackerInput struct {
	ID string `path:"id" format:"uuid"`
}

type trackerOutput struct{ Body Tracker }

func parseTrackerID(raw string) (indexers.TrackerID, error) {
	id, err := storage.ParseUUID(raw)
	if err != nil {
		return indexers.TrackerID{}, trackerNotFound
	}
	return indexers.TrackerID{UUID: id}, nil
}

func (h handlers) getTracker(ctx context.Context, by identity.Admin, in *trackerInput) (*trackerPageOutput, error) {
	id, err := parseTrackerID(in.ID)
	if err != nil {
		return nil, err
	}
	p, err := h.trackers().Page(ctx, by, id)
	if err != nil {
		return nil, trackerAnswer(err)
	}
	st := p.Stats
	return &trackerPageOutput{Body: trackerPage{Tracker: trackerOf(p.Summary), Stats: stats{
		Uploaded: st.Uploaded, Downloaded: st.Downloaded, Ratio: st.Ratio.Float(), Torrents: st.Torrents,
		SeedingBytes: st.SeedingBytes, Grabs30d: st.Grabs30d, Owing: st.Owing,
	}}}, nil
}

func (h handlers) trackerReply(ctx context.Context, by identity.Admin, id indexers.TrackerID) (*trackerOutput, error) {
	s, err := h.trackers().Tracker(ctx, by, id)
	if err != nil {
		return nil, trackerAnswer(err)
	}
	return &trackerOutput{Body: trackerOf(s)}, nil
}

type renameTrackerInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Name string `json:"name" minLength:"1" maxLength:"100"`
	}
}

func (h handlers) renameTracker(ctx context.Context, by identity.Admin, in *renameTrackerInput) (*trackerOutput, error) {
	id, err := parseTrackerID(in.ID)
	if err != nil {
		return nil, err
	}
	if err := h.indexers().RenameTracker(ctx, by, id, in.Body.Name); err != nil {
		return nil, trackerAnswer(err)
	}
	return h.trackerReply(ctx, by, id)
}

type mergeTrackerInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Into string `json:"into" format:"uuid" doc:"The Tracker it merges into, which is kept"`
	}
}

func (h handlers) mergeTracker(ctx context.Context, by identity.Admin, in *mergeTrackerInput) (*trackerOutput, error) {
	from, err := parseTrackerID(in.ID)
	if err != nil {
		return nil, err
	}
	into, err := parseTrackerID(in.Body.Into)
	if err != nil {
		return nil, err
	}
	if err := h.indexers().MergeTracker(ctx, by, from, into); err != nil {
		return nil, trackerAnswer(err)
	}
	return h.trackerReply(ctx, by, into)
}

func Routes(r *api.Router, registry Registry) {
	o := r.Owner("trackers", api.Admin)
	h := handlers{registry}

	api.Register(o, api.Admin, api.Op{
		ID: "listTrackers", Method: http.MethodGet, Path: "/trackers", Summary: "List the Trackers and the Indexers that reach each", Status: http.StatusOK,
	}, h.listTrackers)

	api.Register(o, api.Admin, api.Op{
		ID: "getTracker", Method: http.MethodGet, Path: "/trackers/{id}", Summary: "Get a Tracker", Status: http.StatusOK,
		Errors: []*api.Code{trackerNotFound},
	}, h.getTracker)

	api.Register(o, api.Admin, api.Op{
		ID:      "renameTracker",
		Method:  http.MethodPatch,
		Path:    "/trackers/{id}",
		Summary: "Rename a Tracker. Every Indexer and Release that reached it still does.",
		Status:  http.StatusOK,
		Errors:  []*api.Code{trackerNotFound, trackerNameTaken},
	}, h.renameTracker)

	api.Register(o, api.Admin, api.Op{
		ID:      "mergeTracker",
		Method:  http.MethodPost,
		Path:    "/trackers/{id}/merge",
		Summary: "Merge a Tracker into another, which keeps its name and takes its Indexers, Grabs, Torrents, hosts and seeding history. The merged Tracker is deleted.",
		Status:  http.StatusOK,
		Errors:  []*api.Code{trackerNotFound, trackerMergeSelf},
	}, h.mergeTracker)

	api.Register(o, api.Admin, api.Op{
		ID:      "setTrackerRule",
		Method:  http.MethodPut,
		Path:    "/trackers/{id}/rule",
		Summary: "Set a Tracker's seeding rule. Rules its Releases state never replace it.",
		Status:  http.StatusOK,
		Errors:  []*api.Code{trackerNotFound, trackerPublic, seedingValueRefused},
	}, h.setRule)

	api.Register(o, api.Admin, api.Op{
		ID:      "clearTrackerRule",
		Method:  http.MethodDelete,
		Path:    "/trackers/{id}/rule",
		Summary: "Delete the seeding rule an Admin set for a Tracker, so the latest rule its Releases state is in force",
		Status:  http.StatusOK,
		Errors:  []*api.Code{trackerNotFound, trackerPublic},
	}, h.clearRule)

	api.Register(o, api.Admin, api.Op{
		ID:      "setTrackerGoal",
		Method:  http.MethodPut,
		Path:    "/trackers/{id}/goal",
		Summary: "Set a Tracker's own seeding goal. A goal equal to the default goal is deleted, so the Tracker follows the default.",
		Status:  http.StatusOK,
		Errors:  []*api.Code{trackerNotFound, seedingValueRefused},
	}, h.setGoal)

	api.Register(o, api.Admin, api.Op{
		ID:      "clearTrackerGoal",
		Method:  http.MethodDelete,
		Path:    "/trackers/{id}/goal",
		Summary: "Delete a Tracker's own seeding goal, so it follows the default goal",
		Status:  http.StatusOK,
		Errors:  []*api.Code{trackerNotFound},
	}, h.clearGoal)

	api.Register(o, api.Admin, api.Op{
		ID: "getDefaultGoal", Method: http.MethodGet, Path: "/trackers/default-goal", Summary: "Get the seeding goal of every Tracker with none of its own", Status: http.StatusOK,
	}, h.getDefaultGoal)

	api.Register(o, api.Admin, api.Op{
		ID:      "setDefaultGoal",
		Method:  http.MethodPut,
		Path:    "/trackers/default-goal",
		Summary: "Set the seeding goal of every Tracker with none of its own",
		Status:  http.StatusOK,
		Errors:  []*api.Code{seedingValueRefused},
	}, h.setDefaultGoal)

	api.Register(o, api.Admin, api.Op{
		ID: "getSeedingSettings", Method: http.MethodGet, Path: "/trackers/settings", Summary: "Get the seeding settings", Status: http.StatusOK,
	}, h.getSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateSeedingSettings",
		Method:  http.MethodPatch,
		Path:    "/trackers/settings",
		Summary: "Change seeding settings by key, all or none",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.updateSettings)
}
