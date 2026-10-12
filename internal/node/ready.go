package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"reflect"
)

// Listeners holds every address the Node bound, one field per listener. Start
// fails while any field is zero, so a new listener cannot go unreported.
type Listeners struct {
	Control string         `json:"control"`
	Torrent netip.AddrPort `json:"torrent"`
}

func (l Listeners) unbound() []string {
	var names []string
	v := reflect.ValueOf(l)
	for i := range v.NumField() {
		if v.Field(i).IsZero() {
			names = append(names, v.Type().Field(i).Name)
		}
	}
	return names
}

type Ready struct {
	PID       int       `json:"pid"`
	DataDir   string    `json:"data_dir"`
	Version   string    `json:"version"`
	Seed      string    `json:"seed"`
	Listeners Listeners `json:"listeners"`
	Overrides []Key     `json:"overrides"`
}

var ErrNotReadyRecord = errors.New("not the readiness record")

// ParseReady reads the readiness record from one JSON log line.
func ParseReady(line []byte) (Ready, error) {
	var rec struct {
		Msg   string `json:"msg"`
		Ready *Ready `json:"ready"`
	}
	if err := json.Unmarshal(line, &rec); err != nil {
		return Ready{}, fmt.Errorf("parse log record: %w", err)
	}
	if rec.Msg != "ready" || rec.Ready == nil {
		return Ready{}, ErrNotReadyRecord
	}
	return *rec.Ready, nil
}
