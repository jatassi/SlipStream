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

// RecordFile is the pairing record in the data dir. A record no
// Controller removed makes a Node a Worker.
const RecordFile = "pairing.json"

// schema 2 adds removed, and schema 3 each superseded key's addresses,
// proof and toldAt, and refusedUnknownAt. Each reads an older record as
// it was.
const schema = 3

func recordPath(dataDir string) string { return filepath.Join(dataDir, RecordFile) }

// Record is pairing.json: this Node's id, its Controller and every
// Controller key it replaced. ControllerName is the name the Controller
// gave at pairing, which the tray shows.
type Record struct {
	NodeID         role.NodeID
	Controller     link.Target
	ControllerName string
	Superseded     []Superseded
	PairedAt       time.Time
	// Removed is set once the Controller removed this Node, which is then
	// Unclaimed. The rest is the pairing it lost.
	Removed *Removal
	// RefusedUnknownAt is when the pinned Controller first refused this
	// Node's Hello as a key it has no record of, zero since it last
	// paired or was welcomed.
	RefusedUnknownAt time.Time
}

// Removal is when the Controller removed this Node, and its name.
type Removal struct {
	At             time.Time
	ControllerName string
}

// Sentence is what the log says of the removal.
func (r Removal) Sentence() string {
	return "This Node was removed from " + r.ControllerName + " on " + r.At.Local().Format("2 Jan") + ". Join with a new Join Code to use it again."
}

// IsWorker reports whether the pairing record in dataDir makes the Node a
// Worker: it exists, and no Controller removed the Node.
func IsWorker(dataDir string) (bool, error) {
	rec, ok, err := readRecord(recordPath(dataDir))
	return ok && rec.Removed == nil, err
}

// Superseded is a Controller key a re-pair or a restore replaced, with
// the key that replaced it. Addresses is where that Controller was last
// reached. Proof is what the pinned Controller minted to fence it, nil
// when it gave none, and ToldAt when that Controller first answered a
// Tell carrying Proof.
type Superseded struct {
	Key, By   link.Key
	At        time.Time
	Addresses link.Addresses
	Proof     []byte
	ToldAt    time.Time
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
	Schema           int              `json:"schema"`
	NodeID           string           `json:"nodeId"`
	Controller       controllerJSON   `json:"controller"`
	Superseded       []supersededJSON `json:"superseded"`
	PairedAt         time.Time        `json:"pairedAt"`
	Removed          *removalJSON     `json:"removed,omitempty"`
	RefusedUnknownAt *time.Time       `json:"refusedUnknownAt,omitempty"`
}

type removalJSON struct {
	At             time.Time `json:"at"`
	ControllerName string    `json:"controllerName"`
}

type controllerJSON struct {
	Key       link.Key         `json:"key"`
	Name      string           `json:"name"`
	Addresses []netip.AddrPort `json:"addresses"`
}

type supersededJSON struct {
	Key       link.Key         `json:"key"`
	ByKey     link.Key         `json:"byKey"`
	At        time.Time        `json:"at"`
	Addresses []netip.AddrPort `json:"addresses,omitempty"`
	Proof     []byte           `json:"proof,omitempty"`
	ToldAt    *time.Time       `json:"toldAt,omitempty"`
}

// three is whether j carries a field schema 3 added.
func (j recordJSON) three() bool {
	if j.RefusedUnknownAt != nil {
		return true
	}
	for _, s := range j.Superseded {
		if s.Addresses != nil || s.Proof != nil || s.ToldAt != nil {
			return true
		}
	}
	return false
}

// fits refuses a schema this binary doesn't read, and a field the
// record's schema doesn't have.
func (j recordJSON) fits() error {
	switch {
	case j.Schema < 1 || j.Schema > schema:
		return fmt.Errorf("schema %d, want 1 to %d", j.Schema, schema)
	case j.Schema == 1 && j.Removed != nil:
		return errors.New("removed in a schema 1 record")
	case j.Schema < 3 && j.three():
		return fmt.Errorf("a schema 3 field in a schema %d record", j.Schema)
	}
	return nil
}

func (s supersededJSON) decode() (Superseded, error) {
	sup := Superseded{Key: s.Key, By: s.ByKey, At: s.At, Proof: s.Proof}
	if len(s.Addresses) > 0 {
		var err error
		if sup.Addresses, err = link.ParseAddresses(s.Addresses); err != nil {
			return Superseded{}, err
		}
	}
	if s.ToldAt != nil {
		sup.ToldAt = *s.ToldAt
	}
	return sup, nil
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
	if err := j.fits(); err != nil {
		return Record{}, err
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
		NodeID:         role.NodeID{UUID: id},
		Controller:     link.Target{Key: j.Controller.Key, Addresses: addrs},
		ControllerName: j.Controller.Name,
		PairedAt:       j.PairedAt,
	}
	for i, s := range j.Superseded {
		sup, err := s.decode()
		if err != nil {
			return Record{}, fmt.Errorf("superseded %d addresses: %w", i, err)
		}
		rec.Superseded = append(rec.Superseded, sup)
	}
	if r := j.Removed; r != nil {
		rec.Removed = &Removal{At: r.At, ControllerName: r.ControllerName}
	}
	if j.RefusedUnknownAt != nil {
		rec.RefusedUnknownAt = *j.RefusedUnknownAt
	}
	return rec, nil
}

func encode(rec Record) ([]byte, error) {
	j := recordJSON{
		Schema:     schema,
		NodeID:     rec.NodeID.String(),
		Controller: controllerJSON{Key: rec.Controller.Key, Name: rec.ControllerName, Addresses: rec.Controller.Addresses.All()},
		Superseded: []supersededJSON{},
		PairedAt:   rec.PairedAt,
	}
	for _, s := range rec.Superseded {
		sj := supersededJSON{Key: s.Key, ByKey: s.By, At: s.At, Addresses: s.Addresses.All(), Proof: s.Proof}
		if !s.ToldAt.IsZero() {
			sj.ToldAt = &s.ToldAt
		}
		j.Superseded = append(j.Superseded, sj)
	}
	if r := rec.Removed; r != nil {
		j.Removed = &removalJSON{At: r.At, ControllerName: r.ControllerName}
	}
	if !rec.RefusedUnknownAt.IsZero() {
		j.RefusedUnknownAt = &rec.RefusedUnknownAt
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
