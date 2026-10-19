// Package roots owns the library roots: folders on a Node that hold a
// Content Type's Works, one of each type its default, and the root each
// Work lives in.
package roots

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots/internal/db"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("roots", schemaFiles) }

type ID = db.RootID

// Root is a folder on Node that holds Works of ContentType.
type Root struct {
	ID          ID
	ContentType works.TypeID
	Node        role.NodeID
	// Path is the Node's own string, as the Node cleaned it. The Controller
	// never joins, splits or compares it.
	Path string
	// Default is where its type's new Works go.
	Default bool
	// Place counts the root's homes; a re-home bumps it, so facts about an
	// earlier home can't confirm this one.
	Place int64
}

func rootOf(row db.RootsRoot) Root {
	return Root{ID: row.ID, ContentType: row.ContentType, Node: row.NodeID, Path: row.Path, Default: row.IsDefault, Place: row.Place}
}

// State is a root as the Controller sees it now: from the roster first,
// then from its Node's facts.
type State string

const (
	Ready        State = "ready"
	Missing      State = "missing"
	Unmounted    State = "unmounted"
	Foreign      State = "foreign"
	Unwritable   State = "unwritable"
	Unresponsive State = "unresponsive"
	// Checking: its Node holds it but hasn't reported it at its place yet.
	Checking State = "checking"
	// Offline: its Node is out of step.
	Offline State = "offline"
	// WrittenOff: the Admin wrote off its Node's Storage.
	WrittenOff State = "written_off"
)

// View is a root as the Admin lists it.
type View struct {
	Root
	State State
	// VolumeID is the volume its Node last found it ready on, empty before
	// it did at its place.
	VolumeID string
	Works    int64
	Use      Use
}

// Use is the files a root holds, doomed ones counted, and the bytes of the
// live ones.
type Use struct{ Files, Bytes int64 }

type NewRoot struct {
	ContentType works.TypeID
	Node        role.NodeID
	Path        string
	// MakeDefault makes it its type's default; a type's first root is
	// whatever it says.
	MakeDefault bool
}

// Changed names the roots whose row, Works or state changed.
type Changed struct {
	RootIDs []string `json:"rootIds" nullable:"false"`
}

func changedOf(ids []ID) Changed {
	out := Changed{RootIDs: make([]string, len(ids))}
	for i, id := range ids {
		out.RootIDs[i] = id.String()
	}
	return out
}

var ChangedEvent = live.AdminEvent[Changed]("roots.changed")

type Publisher interface {
	Publish(ctx context.Context, v Changed)
}

var (
	ErrNoType     = errors.New("no Content Type has that id")
	ErrNoRoot     = errors.New("the Work has no root")
	ErrNotFound   = errors.New("no root has that id")
	ErrNotStorage = errors.New("the Node doesn't hold Storage")
	ErrNodeAway   = errors.New("the Node is offline")
	ErrWrongType  = errors.New("the root holds another Content Type's Works")
	ErrHasFiles   = errors.New("the Work has files, so changing its root is a Move")
)

// Reason is why a folder can't be a root: one of storing's reasons, or
// NodeOffline.
type Reason string

const NodeOffline Reason = "node_offline"

// Refusal is why a folder can't be a root, and the root it is inside,
// contains or carries the marker of.
type Refusal struct {
	Reason Reason
	Root   *ID
}

func refusalOf(r storing.Refusal) *Refusal {
	out := &Refusal{Reason: Reason(r.Reason.String())}
	if !r.Root.IsZero() {
		out.Root = &ID{UUID: r.Root}
	}
	return out
}

// UnusableError is a folder that can't be a root, and why.
type UnusableError struct{ Refusal Refusal }

func (e *UnusableError) Error() string {
	return "the folder can't be a root: " + string(e.Refusal.Reason)
}

// Verdict is a folder's fitness as a root, judged on its Node. Refusal is
// nil when it can be one.
type Verdict struct {
	// Path is the folder as its Node cleaned it.
	Path    string
	Volume  *storing.Volume
	Refusal *Refusal
}

// Contents is what files knows about roots. files.RootContents implements
// it.
type Contents interface {
	// Use counts the files of each of ids and their bytes.
	Use(ctx context.Context, r storage.Reader[file], ids []ID) (map[ID]Use, error)
	// HasFiles reports whether work has a file.
	HasFiles(ctx context.Context, r storage.Reader[file], work works.WorkID) (bool, error)
	// Proof is up to three root-relative paths where root's files record
	// their bytes, none while it has no files.
	Proof(ctx context.Context, r storage.Reader[file], root ID) ([]string, error)
}

// Roster is the roster as roots reads it.
type Roster interface {
	Holds(r role.Name, node role.NodeID) bool
	InStep(node role.NodeID) bool
}

// Storage is the Storage Role's *role.Control.
type Storage interface {
	Touch(nodes ...role.NodeID)
	Facts(node role.NodeID) (*storagev1.Facts, bool)
}

type Config struct {
	Library *storage.DB[file]
	// Types is every Content Type this binary registers.
	Types    []works.TypeID
	Roster   Roster
	Storage  Storage
	Remote   storing.Remote
	Arrivals *Arrivals
	Contents Contents
	Changed  Publisher
	Log      *slog.Logger
}

type Roots struct {
	cfg Config
	// mu orders the checks and writes that create roots, so two can't
	// claim one folder.
	mu sync.Mutex
	// last is what each root's Node last reported of it, and at which
	// place. Only Run reads and writes it.
	last map[ID]placed
	// watch is closed and replaced when a report moves.
	watch struct {
		sync.Mutex
		ch chan struct{}
	}
}

// Open homes, in one Write, every rootless Work whose type has a default
// root, as the first root of a type does, and publishes only when it
// wrote.
func Open(ctx context.Context, cfg Config) (*Roots, error) {
	var homed []ID
	err := cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		homed, err = homeRootless(ctx, tx)
		return err
	})
	if err != nil {
		return nil, err
	}
	if len(homed) > 0 {
		cfg.Changed.Publish(ctx, changedOf(homed))
	}
	r := &Roots{cfg: cfg, last: map[ID]placed{}}
	r.watch.ch = make(chan struct{})
	return r, nil
}

// Create makes a folder on n's Node a root of n's type once the Node, which
// must hold Storage, judges it usable apart from the Node's other roots. A
// type's first root is its default and homes the type's rootless Works in
// the same Write.
func (r *Roots) Create(ctx context.Context, _ identity.Admin, n NewRoot) (View, error) {
	if !slices.Contains(r.cfg.Types, n.ContentType) {
		return View{}, ErrNoType
	}
	if !r.cfg.Roster.Holds(role.Storage, n.Node) {
		return View{}, ErrNotStorage
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	v, err := r.check(ctx, n.Node, n.Path, nil)
	if err != nil {
		return View{}, err
	}
	if v.Refusal != nil {
		return View{}, &UnusableError{Refusal: *v.Refusal}
	}
	root := Root{ID: ID{UUID: storage.NewUUID()}, ContentType: n.ContentType, Node: n.Node, Path: v.Path}
	var view View
	err = r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := insert(ctx, tx, root, n.MakeDefault); err != nil {
			return err
		}
		made, err := Get(ctx, tx, root.ID)
		if err != nil {
			return err
		}
		views, err := r.views(ctx, tx, []Root{made})
		if err != nil {
			return err
		}
		view = views[0]
		return nil
	})
	if err != nil {
		return View{}, err
	}
	r.cfg.Changed.Publish(ctx, changedOf([]ID{root.ID}))
	r.cfg.Storage.Touch(root.Node)
	return view, nil
}

// insert saves root, its type's default when makeDefault says so or it is
// the type's first, which also homes the type's rootless Works.
func insert(ctx context.Context, tx *storage.Tx[file], root Root, makeDefault bool) error {
	q := db.New(tx)
	others, err := q.CountRootsOf(ctx, root.ContentType)
	if err != nil {
		return err
	}
	isDefault := others == 0 || makeDefault
	if isDefault {
		if err := q.ClearDefaultRoot(ctx, root.ContentType); err != nil {
			return err
		}
	}
	err = q.InsertRoot(ctx, db.InsertRootParams{ID: root.ID, ContentType: root.ContentType, NodeID: root.Node, Path: root.Path, IsDefault: isDefault})
	if err != nil || others > 0 {
		return err
	}
	_, err = homeRootless(ctx, tx)
	return err
}

// Check judges path on node, which must hold Storage, as a root's folder
// apart from every other root but except, without saving anything.
func (r *Roots) Check(ctx context.Context, _ identity.Admin, node role.NodeID, path string, except *ID) (Verdict, error) {
	if !r.cfg.Roster.Holds(role.Storage, node) {
		return Verdict{}, ErrNotStorage
	}
	return r.check(ctx, node, path, except)
}

// check asks node to judge path apart from node's roots but except. A
// folder whose marker names another root is inside that root, whichever
// Node holds it; an offline Node's verdict is NodeOffline.
func (r *Roots) check(ctx context.Context, node role.NodeID, path string, except *ID) (Verdict, error) {
	var others []storing.Other
	known := map[storage.UUID]bool{}
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		all, err := All(ctx, rd)
		if err != nil {
			return err
		}
		for _, root := range all {
			if except != nil && root.ID == *except {
				continue
			}
			known[root.ID.UUID] = true
			if root.Node == node {
				others = append(others, storing.Other{ID: root.ID.UUID, Path: root.Path})
			}
		}
		return nil
	})
	if err != nil {
		return Verdict{}, err
	}
	c, err := r.cfg.Remote.Check(ctx, node, path, others)
	switch err := reached(err); {
	case errors.Is(err, ErrNodeAway):
		return Verdict{Path: path, Refusal: &Refusal{Reason: NodeOffline}}, nil
	case err != nil:
		return Verdict{}, err
	}
	v := Verdict{Path: c.Path, Volume: c.Volume}
	switch {
	case c.Refusal != nil:
		v.Refusal = refusalOf(*c.Refusal)
	case known[c.Marker]:
		v.Refusal = refusalOf(storing.Refusal{Reason: storing.ReasonInsideRoot, Root: c.Marker})
	}
	return v, nil
}

// reached is a Storage call's error as roots reads it: a Node that doesn't
// hold Storage is ErrNotStorage, and one the call never reached, or lost on
// the way, ErrNodeAway.
func reached(err error) error {
	if why, ok := rpc.NotSent(err); ok {
		if why == rpc.NotHolder {
			return fmt.Errorf("%w: %w", ErrNotStorage, err)
		}
		return fmt.Errorf("%w: %w", ErrNodeAway, err)
	}
	if err != nil && rpc.Lost(err) {
		return fmt.Errorf("%w: %w", ErrNodeAway, err)
	}
	return err
}

// MakeDefault makes root id where its type's new Works go.
func (r *Roots) MakeDefault(ctx context.Context, _ identity.Admin, id ID) error {
	changed := []ID{id}
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		root, err := Get(ctx, tx, id)
		if err != nil {
			return err
		}
		q := db.New(tx)
		was, err := found(q.GetDefaultRoot(ctx, root.ContentType))
		switch {
		case err == nil && was.ID != id:
			changed = append(changed, was.ID)
		case err != nil && !errors.Is(err, ErrNotFound):
			return err
		}
		if err := q.ClearDefaultRoot(ctx, root.ContentType); err != nil {
			return err
		}
		return q.SetDefaultRoot(ctx, id)
	})
	if err != nil {
		return err
	}
	r.cfg.Changed.Publish(ctx, changedOf(changed))
	return nil
}

// List lists every root by Content Type, oldest first in each.
func (r *Roots) List(ctx context.Context, _ identity.Admin) ([]View, error) {
	var out []View
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		all, err := All(ctx, rd)
		if err != nil {
			return err
		}
		out, err = r.views(ctx, rd, all)
		return err
	})
	return out, err
}

func (r *Roots) views(ctx context.Context, rd storage.Reader[file], roots []Root) ([]View, error) {
	q := db.New(rd)
	counts, err := q.CountWorksByRoot(ctx)
	if err != nil {
		return nil, err
	}
	writtenOff, err := nodes.WrittenOffIn(ctx, rd, role.Storage)
	if err != nil {
		return nil, err
	}
	seenRows, err := q.ListSeen(ctx)
	if err != nil {
		return nil, err
	}
	seen := make(map[ID]db.RootsSeen, len(seenRows))
	for _, row := range seenRows {
		seen[row.RootID] = row
	}
	facts := map[role.NodeID]map[ID]placed{}
	homed := make(map[ID]int64, len(counts))
	for _, c := range counts {
		homed[c.RootID] = c.Works
	}
	ids := make([]ID, len(roots))
	for i, root := range roots {
		ids[i] = root.ID
	}
	use, err := r.cfg.Contents.Use(ctx, rd, ids)
	if err != nil {
		return nil, err
	}
	out := make([]View, len(roots))
	for i, root := range roots {
		of, ok := facts[root.Node]
		if !ok {
			if of, err = r.reportedOn(root.Node); err != nil {
				return nil, err
			}
			facts[root.Node] = of
		}
		v := View{Root: root, Works: homed[root.ID], Use: use[root.ID]}
		var rep report
		if p, ok := of[root.ID]; ok && p.place == root.Place {
			rep = p.report
		}
		v.State = stateOf(writtenOff[root.Node], r.cfg.Roster.InStep(root.Node), rep.state)
		if s, ok := seen[root.ID]; ok && s.Place == root.Place {
			v.VolumeID = string(s.VolumeID)
		}
		out[i] = v
	}
	return out, nil
}

// All lists every root in r by Content Type, oldest first in each.
func All(ctx context.Context, r storage.Reader[file]) ([]Root, error) {
	rows, err := db.New(r).ListRoots(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Root, len(rows))
	for i, row := range rows {
		out[i] = rootOf(row)
	}
	return out, nil
}

// Get reads root id in r, ErrNotFound when no root has it.
func Get(ctx context.Context, r storage.Reader[file], id ID) (Root, error) {
	return found(db.New(r).GetRoot(ctx, id))
}

// OfWork reads the root work lives in, ErrNoRoot while it is rootless.
func OfWork(ctx context.Context, r storage.Reader[file], work works.WorkID) (Root, error) {
	row, err := db.New(r).GetRootOfWork(ctx, work)
	if errors.Is(err, storage.ErrNoRows) {
		return Root{}, ErrNoRoot
	}
	if err != nil {
		return Root{}, err
	}
	return rootOf(row), nil
}

func found(row db.RootsRoot, err error) (Root, error) {
	if errors.Is(err, storage.ErrNoRows) {
		return Root{}, ErrNotFound
	}
	if err != nil {
		return Root{}, err
	}
	return rootOf(row), nil
}
