package indexersapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	trackerNotFound  = api.NewCode(http.StatusNotFound, "tracker_not_found", "No Tracker has that id")
	trackerNameTaken = api.NewCode(http.StatusConflict, "tracker_name_taken", "Another Tracker has that name")
	trackerMergeSelf = api.NewCode(http.StatusUnprocessableEntity, "tracker_merge_self", "A Tracker can't be merged into itself")
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

type tracker struct {
	ID         string   `json:"id" format:"uuid"`
	Name       string   `json:"name"`
	Privacy    string   `json:"privacy" enum:"private,semi-private,public,unknown"`
	Hosts      []string `json:"hosts" nullable:"false" doc:"The announce hosts of its private torrents"`
	Keys       []string `json:"keys" nullable:"false" doc:"How a Release finds it: definition:<id>, prowlarr:<feed Indexer>:<source id> or named:<Indexer>"`
	IndexerIDs []string `json:"indexerIds" nullable:"false" doc:"The Indexers that reach it, by name"`
}

func trackerOf(t indexers.Tracker) tracker {
	out := tracker{
		ID: t.ID.String(), Name: t.Name, Privacy: string(t.Privacy),
		Hosts: append([]string{}, t.Hosts...), Keys: append([]string{}, t.Keys...), IndexerIDs: []string{},
	}
	for _, id := range t.Indexers {
		out.IndexerIDs = append(out.IndexerIDs, id.String())
	}
	return out
}

type trackerList struct {
	Trackers []tracker `json:"trackers" nullable:"false" doc:"By name"`
}

type trackerListOutput struct{ Body trackerList }

func (h handlers) listTrackers(ctx context.Context, by identity.Admin, _ *struct{}) (*trackerListOutput, error) {
	list, err := open(h.registry).Trackers(ctx, by)
	if err != nil {
		return nil, err
	}
	out := trackerList{Trackers: []tracker{}}
	for _, t := range list {
		out.Trackers = append(out.Trackers, trackerOf(t))
	}
	return &trackerListOutput{Body: out}, nil
}

type trackerInput struct {
	ID string `path:"id" format:"uuid"`
}

type trackerOutput struct{ Body tracker }

func parseTrackerID(raw string) (indexers.TrackerID, error) {
	id, err := storage.ParseUUID(raw)
	if err != nil {
		return indexers.TrackerID{}, trackerNotFound
	}
	return indexers.TrackerID{UUID: id}, nil
}

func (h handlers) getTracker(ctx context.Context, by identity.Admin, in *trackerInput) (*trackerOutput, error) {
	id, err := parseTrackerID(in.ID)
	if err != nil {
		return nil, err
	}
	return h.trackerReply(ctx, by, id)
}

func (h handlers) trackerReply(ctx context.Context, by identity.Admin, id indexers.TrackerID) (*trackerOutput, error) {
	t, err := open(h.registry).Tracker(ctx, by, id)
	if err != nil {
		return nil, trackerAnswer(err)
	}
	return &trackerOutput{Body: trackerOf(t)}, nil
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
	if err := open(h.registry).RenameTracker(ctx, by, id, in.Body.Name); err != nil {
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
	if err := open(h.registry).MergeTracker(ctx, by, from, into); err != nil {
		return nil, trackerAnswer(err)
	}
	return h.trackerReply(ctx, by, into)
}
