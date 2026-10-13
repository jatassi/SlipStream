package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/storage"
)

type FileID struct{ storage.UUID }

// Edition is a cut of an Item. Every Edition is Standard until Editions
// arrive.
type Edition string

const Standard Edition = "standard"

// Probe is what probing found in a file.
type Probe struct {
	Width  int `json:"width"`
	Height int `json:"height"`
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
	return json.Unmarshal([]byte(s), p)
}
