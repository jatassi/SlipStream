package subtitles

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
)

// SidecarsOf reads, in r, the Sidecars of each of fs that has one.
func SidecarsOf(ctx context.Context, r storage.Reader[storage.Library], fs []files.ID) (map[files.ID][]Sidecar, error) {
	listed, err := json.Marshal(fs)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	rows, err := db.New(r).ListSidecarsOfFiles(ctx, string(listed))
	if err != nil {
		return nil, err
	}
	out := map[files.ID][]Sidecar{}
	for _, row := range rows {
		s, err := sidecarOf(row)
		if err != nil {
			return nil, err
		}
		out[row.FileID] = append(out[row.FileID], s)
	}
	return out, nil
}
