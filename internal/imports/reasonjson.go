package imports

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// The id and time types keep their values unexported, so encoding/json
// would write each as {}. A Reason and a Hold are stored with their ids
// as strings and Since in Unix milliseconds.

type reasonAlias Reason

type reasonJSON struct {
	reasonAlias
	Targets []string `json:"targets,omitempty"`
	Reading []string `json:"reading,omitempty"`
	Of      []string `json:"of,omitempty"`
	Level   string   `json:"level,omitempty"`
}

func (r Reason) MarshalJSON() ([]byte, error) {
	out := reasonJSON{reasonAlias: reasonAlias(r), Targets: idStrings(r.Targets), Reading: idStrings(r.Reading), Of: idStrings(r.Of)}
	if !r.Level.IsZero() {
		out.Level = r.Level.String()
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("encode the reason: %w", err)
	}
	return b, nil
}

func (r *Reason) UnmarshalJSON(b []byte) error {
	var in reasonJSON
	if err := json.Unmarshal(b, &in); err != nil {
		return fmt.Errorf("decode the reason: %w", err)
	}
	*r = Reason(in.reasonAlias)
	var err error
	if r.Targets, err = parseIDs(in.Targets, func(u storage.UUID) targets.TargetID { return targets.TargetID{UUID: u} }); err != nil {
		return err
	}
	if r.Reading, err = parseIDs(in.Reading, func(u storage.UUID) works.ItemID { return works.ItemID{UUID: u} }); err != nil {
		return err
	}
	if r.Of, err = parseIDs(in.Of, func(u storage.UUID) works.ItemID { return works.ItemID{UUID: u} }); err != nil {
		return err
	}
	if in.Level != "" {
		u, err := storage.ParseUUID(in.Level)
		if err != nil {
			return err
		}
		r.Level = works.LevelID{UUID: u}
	}
	return nil
}

func idStrings[T interface{ String() string }](ids []T) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

func parseIDs[T any](ss []string, of func(storage.UUID) T) ([]T, error) {
	if len(ss) == 0 {
		return nil, nil
	}
	out := make([]T, len(ss))
	for i, s := range ss {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return nil, err
		}
		out[i] = of(u)
	}
	return out, nil
}

type holdAlias Hold

type holdJSON struct {
	holdAlias
	Since int64 `json:"since"`
}

func (h Hold) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(holdJSON{holdAlias: holdAlias(h), Since: h.Since.Time().UnixMilli()})
	if err != nil {
		return nil, fmt.Errorf("encode the hold: %w", err)
	}
	return b, nil
}

func (h *Hold) UnmarshalJSON(b []byte) error {
	var in holdJSON
	if err := json.Unmarshal(b, &in); err != nil {
		return fmt.Errorf("decode the hold: %w", err)
	}
	*h = Hold(in.holdAlias)
	h.Since = storage.FromTime(time.UnixMilli(in.Since))
	return nil
}
