package indexersapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
)

type tracker struct {
	ID         string   `json:"id" format:"uuid"`
	Name       string   `json:"name"`
	Privacy    string   `json:"privacy" enum:"private,semi-private,public,unknown"`
	IndexerIDs []string `json:"indexerIds" nullable:"false" doc:"The Indexers that reach it, by name"`
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
		row := tracker{ID: t.ID.String(), Name: t.Name, Privacy: string(t.Privacy), IndexerIDs: []string{}}
		for _, id := range t.Indexers {
			row.IndexerIDs = append(row.IndexerIDs, id.String())
		}
		out.Trackers = append(out.Trackers, row)
	}
	return &trackerListOutput{Body: out}, nil
}
