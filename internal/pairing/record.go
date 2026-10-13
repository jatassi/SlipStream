package pairing

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/netip"
	"os"
	"path/filepath"
	"time"

	"github.com/jatassi/slipstream/internal/atomicfile"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// RecordFile is the pairing record in the data dir. Its existence makes a
// Node a Worker.
const RecordFile = "pairing.json"

const schema = 1

func recordPath(dataDir string) string { return filepath.Join(dataDir, RecordFile) }

// Record is pairing.json: this Node's id, its Controller and every
// Controller key it replaced.
type Record struct {
	NodeID     role.NodeID
	Controller link.Target
	Superseded []Superseded
	PairedAt   time.Time
}

// Superseded is a Controller key a re-pair replaced, with the key that
// replaced it.
type Superseded struct {
	Key, By link.Key
	At      time.Time
}

// RecordError is a pairing.json this binary cannot use.
type RecordError struct {
	Path string
	Err  error
}

func (e *RecordError) Error() string { return fmt.Sprintf("pairing record %s: %v", e.Path, e.Err) }

func (e *RecordError) Unwrap() error { return e.Err }

// recordJSON only gains fields across versions; decoding refuses the
// rest.
type recordJSON struct {
	Schema     int              `json:"schema"`
	NodeID     string           `json:"nodeId"`
	Controller controllerJSON   `json:"controller"`
	Superseded []supersededJSON `json:"superseded"`
	PairedAt   time.Time        `json:"pairedAt"`
}

type controllerJSON struct {
	Key       link.Key         `json:"key"`
	Addresses []netip.AddrPort `json:"addresses"`
}

type supersededJSON struct {
	Key   link.Key  `json:"key"`
	ByKey link.Key  `json:"byKey"`
	At    time.Time `json:"at"`
}

func readRecord(path string) (Record, bool, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is the data dir's pairing record, chosen by the Node
	if errors.Is(err, fs.ErrNotExist) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, &RecordError{Path: path, Err: err}
	}
	rec, err := decode(data)
	if err != nil {
		return Record{}, false, &RecordError{Path: path, Err: err}
	}
	return rec, true, nil
}

func decode(data []byte) (Record, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var j recordJSON
	if err := dec.Decode(&j); err != nil {
		return Record{}, fmt.Errorf("decode: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return Record{}, errors.New("data after the record")
	}
	if j.Schema != schema {
		return Record{}, fmt.Errorf("schema %d, want %d", j.Schema, schema)
	}
	id, err := storage.ParseUUID(j.NodeID)
	if err != nil {
		return Record{}, fmt.Errorf("nodeId: %w", err)
	}
	var zero link.Key
	if j.Controller.Key == zero {
		return Record{}, errors.New("no controller.key")
	}
	addrs, err := link.ParseAddresses(j.Controller.Addresses)
	if err != nil {
		return Record{}, fmt.Errorf("controller.addresses: %w", err)
	}
	rec := Record{
		NodeID:     role.NodeID{UUID: id},
		Controller: link.Target{Key: j.Controller.Key, Addresses: addrs},
		PairedAt:   j.PairedAt,
	}
	for _, s := range j.Superseded {
		rec.Superseded = append(rec.Superseded, Superseded{Key: s.Key, By: s.ByKey, At: s.At})
	}
	return rec, nil
}

func encode(rec Record) ([]byte, error) {
	j := recordJSON{
		Schema:     schema,
		NodeID:     rec.NodeID.String(),
		Controller: controllerJSON{Key: rec.Controller.Key, Addresses: rec.Controller.Addresses.All()},
		Superseded: []supersededJSON{},
		PairedAt:   rec.PairedAt,
	}
	for _, s := range rec.Superseded {
		j.Superseded = append(j.Superseded, supersededJSON{Key: s.Key, ByKey: s.By, At: s.At})
	}
	data, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	return append(data, '\n'), nil
}

// writeRecord replaces pairing.json, mode 0600, by temp file, fsync and
// rename. Only this package writes it.
func writeRecord(path string, rec Record) error {
	data, err := encode(rec)
	if err != nil {
		return fmt.Errorf("encode the pairing record: %w", err)
	}
	if err := atomicfile.Write(path, data); err != nil {
		return fmt.Errorf("write the pairing record: %w", err)
	}
	return nil
}
