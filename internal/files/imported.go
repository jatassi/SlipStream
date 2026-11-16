package files

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Replaced is a library file a landing doomed, as its files_imported row
// keeps it.
type Replaced struct {
	Quality release.Quality `json:"quality"`
	Label   string          `json:"label"`
	Items   []works.ItemID  `json:"items"`
	Path    string          `json:"path"`
}

// ImportedIn is what each file import placed into work replaced, by file,
// on the files_imported_work index. A file deleted since is left out.
func ImportedIn(ctx context.Context, r storage.Reader[file], work works.WorkID, imp storage.UUID) (map[ID][]Replaced, error) {
	rows, err := db.New(r).ListImportedIn(ctx, db.ListImportedInParams{WorkID: work, ImportID: imp})
	if err != nil {
		return nil, err
	}
	out := make(map[ID][]Replaced, len(rows))
	for _, row := range rows {
		if row.FileID.IsZero() {
			continue
		}
		var replaced []Replaced
		if err := json.Unmarshal([]byte(row.Replaced), &replaced); err != nil {
			return nil, fmt.Errorf("file %s: decode its import record: %w", row.FileID, err)
		}
		out[row.FileID] = replaced
	}
	return out, nil
}
