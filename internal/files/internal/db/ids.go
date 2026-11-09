package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
)

type FileID struct{ storage.UUID }

// Probe is what probing found in a file: its video's frame and codec,
// the codec by ffprobe's name.
type Probe struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Codec  string `json:"codec"`
}

func (p Probe) Value() (driver.Value, error) {
	b, err := json.Marshal(p)
	return string(b), err
}

func (p *Probe) Scan(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("scan a probe from %T", v)
	}
	if err := json.Unmarshal([]byte(s), p); err != nil {
		return fmt.Errorf("scan a probe: %w", err)
	}
	return nil
}

// Reading is a file's release name as release.Read read it, kept as JSON.
type Reading struct{ release.Read }

func (r Reading) Value() (driver.Value, error) {
	b, err := json.Marshal(r.Read)
	return string(b), err
}

func (r *Reading) Scan(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("scan a title read from %T", v)
	}
	if err := json.Unmarshal([]byte(s), &r.Read); err != nil {
		return fmt.Errorf("scan a title read: %w", err)
	}
	return nil
}
