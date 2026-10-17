// Package nodes owns a Controller's Nodes: library set nodes, the roster,
// the Controller's Trust and Pair handler, Join Code minting, the Join Code
// lifetime setting, the facts view and nodes.changed.
package nodes

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes/internal/db"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("nodes", schemaFiles) }

// ID is a Node's id, the NodeID that intent, results and calls name.
type ID = role.NodeID

// Changed tells Admins that a Node's row, link or facts changed.
type Changed struct {
	NodeID string `json:"nodeId" format:"uuid"`
}

var ChangedEvent = live.AdminEvent[Changed]("nodes.changed")

// Publisher is ChangedEvent bound to the hub.
type Publisher interface {
	Publish(ctx context.Context, v Changed)
}

// SettingsChanged tells Admins that a Node's settings view changed: a
// save, or a restart that adopted a pending value.
type SettingsChanged struct {
	NodeID string `json:"nodeId" format:"uuid"`
}

var SettingsChangedEvent = live.AdminEvent[SettingsChanged]("node.settings-changed")

// SettingsPublisher is SettingsChangedEvent bound to the hub.
type SettingsPublisher interface {
	Publish(ctx context.Context, v SettingsChanged)
}

// Link is the Node link as the roster reads it.
type Link interface {
	State(key link.Key) link.State
	Subscribe(ctx context.Context) <-chan link.Change
	Advertised() (link.Addresses, bool)
}

type Config struct {
	Library *storage.DB[file]
	// Key and Name are this Node's. Name is its row's name when Open
	// creates the row.
	Key  link.Key
	Name string
	// Node is this Node's build, shown on its own row.
	Node    link.NodeInfo
	Link    Link
	Changed Publisher
	// SettingsChanged is published whenever a Node's facts carry a new
	// settings revision.
	SettingsChanged SettingsPublisher
	// Served is every Role's control half.
	Served []role.Served
	// Owners keeps each Role's work: exactly one per Role of the table.
	Owners []Owner
	// Always is Roles every Node holds beyond its nodes_role rows: only a
	// test's, as the simulation tier runs roletest's Probe.
	Always  []role.Name
	Timings Timings
	Log     *slog.Logger
}

var (
	ErrNotFound      = errors.New("no Node has this id")
	ErrNoLinkAddress = errors.New("the link has no address to advertise")
)

var JoinCodeLifetime = setting.Duration{
	Key:     "nodes.joinCodeLifetime",
	Label:   "Join Code lifetime",
	Default: time.Hour,
	Min:     10 * time.Minute,
	Max:     7 * 24 * time.Hour,
	Step:    time.Minute,
}

// Service is the Controller's roster and its Nodes' records. Its memory
// holds what calls read on every request: each Node's key, and the board.
type Service struct {
	cfg      Config
	self     ID
	name     string
	opened   time.Time
	owners   map[role.Name]Owner
	controls *role.Controls

	mu    sync.Mutex
	keys  map[ID]link.Key
	byKey map[link.Key]ID
	wake  map[ID]chan struct{}
	// revisions is each Node's last settings revision. It outlives a
	// session, so a Node that reconnects unchanged publishes nothing.
	revisions map[ID]string

	// rmu guards in and saved. board is derived from in, and replaced
	// whole, so reads never lock.
	rmu   sync.Mutex
	in    inputs
	saved map[ID]time.Time
	board atomic.Pointer[Board]
	rearm chan struct{}
}

// Open creates this Node's own row when it has none, holding every Role
// but Relaying, loads the roster and builds the control half. It panics
// unless every Role has exactly one owner.
func Open(ctx context.Context, cfg Config) (*Service, error) {
	s := &Service{
		cfg:    cfg,
		opened: time.Now(),
		owners: ownersByRole(cfg.Owners),
		keys:   map[ID]link.Key{},
		byKey:  map[link.Key]ID{},
		wake:   map[ID]chan struct{}{},

		revisions: map[ID]string{},
		saved:     map[ID]time.Time{},
		rearm:     make(chan struct{}, 1),
	}
	err := cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		row, err := q.GetNodeByKey(ctx, cfg.Key[:])
		if err == nil {
			s.self, s.name = row.ID, row.Name
			return nil
		}
		if !errors.Is(err, storage.ErrNoRows) {
			return err
		}
		now := storage.Now()
		s.self = ID{UUID: storage.NewUUID()}
		if s.name, err = freeName(ctx, q, cfg.Name); err != nil {
			return err
		}
		if err := q.InsertNode(ctx, db.InsertNodeParams{ID: s.self, PublicKey: cfg.Key[:], Name: s.name, PairedAt: now}); err != nil {
			return err
		}
		for _, r := range role.Names() {
			if r == role.Relaying {
				continue
			}
			if err := q.InsertRole(ctx, db.InsertRoleParams{NodeID: s.self, Role: r, AssignedAt: now}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("open the roster: %w", err)
	}
	if err := s.load(ctx); err != nil {
		return nil, err
	}
	rows, grace, err := s.read(ctx)
	if err != nil {
		return nil, err
	}
	s.in = inputs{
		controller: s.self,
		start:      s.opened,
		grace:      grace,
		blip:       cfg.Timings.Blip,
		settle:     cfg.Timings.Settle,
		nodes:      rows,
	}
	s.board.Store(derive(s.in, s.opened))
	s.controls = role.NewControls(role.ControlsConfig{
		Library: cfg.Library,
		Nodes:   s,
		Scope:   s.scope,
		Changed: s.changed,
		InStep:  s.inStep,
		Seen:    s.seen,
		Log:     cfg.Log,
	}, cfg.Served...)
	return s, nil
}

func (s *Service) load(ctx context.Context) error {
	q := db.New(s.cfg.Library)
	rows, err := q.ListNodes(ctx)
	if err != nil {
		return fmt.Errorf("load the roster: %w", err)
	}
	for _, row := range rows {
		key, err := link.KeyFromBytes(row.PublicKey)
		if err != nil {
			return fmt.Errorf("load the roster: node %s: %w", row.ID, err)
		}
		s.keys[row.ID] = key
		s.byKey[key] = row.ID
	}
	return nil
}

// Controls is the control half, serving every Role over this roster.
func (s *Service) Controls() *role.Controls { return s.controls }

func (s *Service) Self() ID { return s.self }

func (s *Service) NodeOf(key link.Key) (ID, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byKey[key]
	return id, ok
}

func (s *Service) Key(node ID) (link.Key, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.keys[node]
	return key, ok
}

// Admit pins every Worker in the roster and lets any other key only Pair.
func (s *Service) Admit(key link.Key) link.Admission {
	if key == s.cfg.Key {
		return link.Refused{}
	}
	if _, ok := s.NodeOf(key); ok {
		return link.Pinned{}
	}
	return link.Pairing{}
}

// assigned is the Roles assigned to node, in table order, then Always.
func (s *Service) assigned(ctx context.Context, r storage.Reader[file], node ID) ([]role.Name, error) {
	roles, err := db.New(r).NodeRoles(ctx, node)
	if err != nil {
		return nil, err
	}
	return append(role.Of(roles...).Names(), s.cfg.Always...), nil
}

// freeName is name, or name-2, name-3 and on when another Node holds it.
func freeName(ctx context.Context, q *db.Queries, name string) (string, error) {
	if name == "" {
		name = "node"
	}
	for n := 1; ; n++ {
		candidate := name
		if n > 1 {
			candidate = name + "-" + strconv.Itoa(n)
		}
		taken, err := q.NameTaken(ctx, candidate)
		if err != nil || !taken {
			return candidate, err
		}
	}
}

// Pair accepts a Join Code's secret from peer in one Write: it finds the
// code by the secret's hash, refuses an unknown, expired or used one,
// upserts peer's row by its key and marks the code used. A known key keeps
// its id, name and Roles, and the key that used a code may use it again.
// Peer is pinned before Pair returns, so its Hello is welcome.
func (s *Service) Pair(ctx context.Context, peer link.Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	hash := sha256.Sum256(req.GetSecret())
	var (
		row     db.NodesNode
		refused linkv1.RefusedReason
	)
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		row, refused, err = pairIn(ctx, db.New(tx), peer, hash[:], req.GetName())
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("pair %s: %w", peer.Fingerprint(), err)
	}
	if refused != linkv1.RefusedReason_REFUSED_REASON_UNSPECIFIED {
		return &linkv1.PairResponse{Outcome: &linkv1.PairResponse_Refused{Refused: &linkv1.Refused{Reason: refused}}}, nil
	}
	s.mu.Lock()
	if _, ok := s.keys[row.ID]; !ok {
		s.keys[row.ID] = peer
		s.byKey[peer] = row.ID
	}
	s.mu.Unlock()
	if err := s.reload(ctx); err != nil {
		return nil, fmt.Errorf("pair %s: %w", peer.Fingerprint(), err)
	}
	s.changed(row.ID)
	return &linkv1.PairResponse{Outcome: &linkv1.PairResponse_Paired{
		Paired: &linkv1.Paired{NodeId: row.ID.Bytes(), Name: row.Name, ControllerName: s.name},
	}}, nil
}

// pairIn is Pair inside its Write: peer's row, or why the code is refused.
func pairIn(ctx context.Context, q *db.Queries, peer link.Key, hash []byte, name string) (db.NodesNode, linkv1.RefusedReason, error) {
	code, err := q.GetJoinCode(ctx, hash)
	if errors.Is(err, storage.ErrNoRows) {
		return db.NodesNode{}, linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_UNKNOWN, nil
	}
	if err != nil {
		return db.NodesNode{}, 0, err
	}
	row, err := q.GetNodeByKey(ctx, peer[:])
	known := err == nil
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		return db.NodesNode{}, 0, err
	}
	now := storage.Now()
	switch {
	case known && code.UsedBy == row.ID:
		return row, 0, nil
	case !code.UsedBy.IsZero():
		return db.NodesNode{}, linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_USED, nil
	case code.ExpiresAt.Compare(now) <= 0:
		return db.NodesNode{}, linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_EXPIRED, nil
	}
	if known {
		err = q.SetPairedAt(ctx, db.SetPairedAtParams{ID: row.ID, PairedAt: now})
	} else {
		row = db.NodesNode{ID: ID{UUID: storage.NewUUID()}, PublicKey: peer[:], PairedAt: now}
		if row.Name, err = freeName(ctx, q, name); err == nil {
			err = q.InsertNode(ctx, db.InsertNodeParams{ID: row.ID, PublicKey: row.PublicKey, Name: row.Name, PairedAt: row.PairedAt})
		}
	}
	if err != nil {
		return db.NodesNode{}, 0, err
	}
	return row, 0, q.UseJoinCode(ctx, db.UseJoinCodeParams{UsedBy: row.ID, UsedAt: now, SecretHash: hash})
}

// JoinCode is a minted code and when the Controller stops accepting it.
type JoinCode struct {
	Code      joincode.Code
	ExpiresAt time.Time
}

// MintJoinCode draws a secret and stores only its hash, expiring after the
// Join Code lifetime. The code names this Node's advertised addresses.
func (s *Service) MintJoinCode(ctx context.Context) (JoinCode, error) {
	addrs, ok := s.cfg.Link.Advertised()
	if !ok {
		return JoinCode{}, ErrNoLinkAddress
	}
	var secret [joincode.SecretSize]byte
	_, _ = rand.Read(secret[:])
	code, err := joincode.New(ed25519.PublicKey(s.cfg.Key[:]), secret, addrs.All())
	if err != nil {
		return JoinCode{}, err
	}
	hash := sha256.Sum256(secret[:])
	var expires storage.Time
	err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		lifetime, err := q.GetJoinCodeLifetime(ctx)
		if err != nil {
			return err
		}
		now := storage.Now()
		expires = storage.FromTime(now.Time().Add(JoinCodeLifetime.Value(lifetime)))
		return q.InsertJoinCode(ctx, db.InsertJoinCodeParams{SecretHash: hash[:], CreatedAt: now, ExpiresAt: expires})
	})
	if err != nil {
		return JoinCode{}, fmt.Errorf("mint a Join Code: %w", err)
	}
	return JoinCode{Code: code, ExpiresAt: expires.Time()}, nil
}

// PruneJoinCodes deletes every Join Code that expired more than one Join
// Code lifetime ago. The grace keeps Pair's answer to a late retry of a
// used code.
func (s *Service) PruneJoinCodes(ctx context.Context) error {
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		lifetime, err := q.GetJoinCodeLifetime(ctx)
		if err != nil {
			return err
		}
		return q.DeleteJoinCodesExpiredBefore(ctx, storage.FromTime(time.Now().Add(-JoinCodeLifetime.Value(lifetime))))
	})
	if err != nil {
		return fmt.Errorf("prune Join Codes: %w", err)
	}
	return nil
}

func (s *Service) JoinCodeLifetimeField(ctx context.Context) (setting.DurationField, error) {
	lifetime, err := db.New(s.cfg.Library).GetJoinCodeLifetime(ctx)
	if err != nil {
		return setting.DurationField{}, fmt.Errorf("read the Join Code lifetime: %w", err)
	}
	return JoinCodeLifetime.Field(lifetime), nil
}

func (s *Service) SetJoinCodeLifetime(ctx context.Context, v setting.Stored[int64]) error {
	return s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).SetJoinCodeLifetime(ctx, v.Override())
	})
}

// SetRoles makes roles the set node holds and tells the control half. For
// a Node in step, here or over a live link, it returns once the Node's
// facts show the node-scope version derived from that set, or ctx ends.
func (s *Service) SetRoles(ctx context.Context, node ID, roles []role.Name) (Node, error) {
	want := role.Of(roles...).Names()
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return replaceRoles(ctx, db.New(tx), node, want)
	})
	if err != nil {
		return Node{}, err
	}
	if err := s.reload(ctx); err != nil {
		return Node{}, err
	}
	s.controls.Touch(node)
	s.changed(node)
	if err := s.AwaitDelivery(ctx, node); err != nil {
		return Node{}, err
	}
	return s.Get(ctx, node)
}

func replaceRoles(ctx context.Context, q *db.Queries, node ID, want []role.Name) error {
	if _, err := q.GetNode(ctx, node); errors.Is(err, storage.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	held, err := q.NodeRoles(ctx, node)
	if err != nil {
		return err
	}
	for _, r := range held {
		if slices.Contains(want, r) {
			continue
		}
		if err := q.DeleteRole(ctx, db.DeleteRoleParams{NodeID: node, Role: r}); err != nil {
			return err
		}
	}
	now := storage.Now()
	for _, r := range want {
		if err := q.InsertRole(ctx, db.InsertRoleParams{NodeID: node, Role: r, AssignedAt: now}); err != nil {
			return err
		}
	}
	return nil
}

// AwaitDelivery waits while node can receive its intent and its facts lag
// the node scope the library derives now. A Node can receive while it is
// in step, or while its link is live and matched and its Sync session is
// about to attach.
func (s *Service) AwaitDelivery(ctx context.Context, node ID) error {
	for {
		woken := s.awaitChange(node)
		if !s.receives(node) {
			return nil
		}
		want, err := s.controls.NodeVersion(ctx, node)
		if err != nil {
			return err
		}
		if f, ok := s.controls.NodeFacts(node); ok && bytes.Equal(f.GetAppliedVersion(), want) {
			return nil
		}
		select {
		case <-woken:
		case <-ctx.Done():
			return fmt.Errorf("await delivery to node %s: %w", node, ctx.Err())
		}
	}
}

func (s *Service) receives(node ID) bool {
	if s.InStep(node) {
		return true
	}
	key, ok := s.Key(node)
	if !ok {
		return false
	}
	up, on := s.cfg.Link.State(key).(link.Live)
	return on && up.Matched
}

// Reach is how the Controller reaches node now and since when: Local for
// itself, Up while its Sync session is attached, Mismatched while its link
// is live on another build, and Down otherwise, since its last Sync frame.
// A Node that can start work is Local or Up.
func (s *Service) Reach(node ID) (LinkState, time.Time) {
	v, ok := s.Board().Node(node)
	switch {
	case !ok:
		return Down, s.opened
	case v.Controller:
		return Local, v.Since
	case v.Standing == InStep:
		return Up, v.Since
	case v.Standing == MismatchedNode:
		return Mismatched, v.Since
	}
	return Down, v.Since
}

func (s *Service) awaitChange(node ID) <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.wake[node]
	if !ok {
		ch = make(chan struct{})
		s.wake[node] = ch
	}
	return ch
}

// changed wakes whoever waits on node and tells Admins, of its settings
// too when its facts carry a new revision. It never blocks.
func (s *Service) changed(node ID) {
	facts, _ := s.controls.NodeFacts(node)
	revision := facts.GetSettingsRevision()
	s.mu.Lock()
	ch := s.wake[node]
	delete(s.wake, node)
	settings := revision != "" && revision != s.revisions[node]
	if settings {
		s.revisions[node] = revision
	}
	s.mu.Unlock()
	if ch != nil {
		close(ch)
	}
	s.cfg.Changed.Publish(context.Background(), Changed{NodeID: node.String()})
	if settings {
		s.cfg.SettingsChanged.Publish(context.Background(), SettingsChanged{NodeID: node.String()})
	}
}

func (s *Service) recordAddresses(ctx context.Context, node ID, addrs []netip.AddrPort) error {
	return s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		held, err := q.NodeAddresses(ctx, node)
		if err != nil {
			return err
		}
		if slices.Equal(held, addressStrings(addrs)) {
			return nil
		}
		if err := q.DeleteAddresses(ctx, node); err != nil {
			return err
		}
		for i, a := range addrs {
			if err := q.InsertAddress(ctx, db.InsertAddressParams{NodeID: node, Position: int64(i), Address: a.String()}); err != nil {
				return err
			}
		}
		return nil
	})
}

func addressStrings(addrs []netip.AddrPort) []string {
	out := make([]string, len(addrs))
	for i, a := range addrs {
		out[i] = a.String()
	}
	return out
}

// LinkState is how the Controller reaches a Node: Local for itself, else
// Up, Down or Mismatched.
type LinkState uint8

const (
	Down LinkState = iota
	Local
	Up
	Mismatched
)

func (l LinkState) String() string {
	return [...]string{"down", "local", "up", "mismatched"}[l]
}

// Node is one Node as an Admin sees it: its row, the node-scope version
// the library derives now, its link and its facts. Facts is nil while it
// has no session.
type Node struct {
	ID        ID
	Name      string
	Key       link.Key
	Self      bool
	PairedAt  time.Time
	Addresses link.Addresses
	Roles     []role.Name
	Version   []byte
	Link      LinkState
	Since     time.Time
	// Build is the peer's, or this Node's own for Local; nil while Down.
	Build *link.NodeInfo
	Facts *nodev1.NodeFacts
}

type Named struct {
	ID   ID
	Name string
}

// Names is every Node's id and name in id order, so the Controller's own
// row, the first paired, comes first.
func Names(ctx context.Context, r storage.Reader[file]) ([]Named, error) {
	rows, err := db.New(r).ListNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list the Nodes: %w", err)
	}
	out := make([]Named, len(rows))
	for i, row := range rows {
		out[i] = Named{ID: row.ID, Name: row.Name}
	}
	return out, nil
}

// List is every Node, in id order.
func (s *Service) List(ctx context.Context) ([]Node, error) {
	q := db.New(s.cfg.Library)
	rows, err := q.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	addrs, err := q.ListAddresses(ctx)
	if err != nil {
		return nil, err
	}
	byNode := map[ID][]string{}
	for _, a := range addrs {
		byNode[a.NodeID] = append(byNode[a.NodeID], a.Address)
	}
	nodes := make([]Node, 0, len(rows))
	for _, row := range rows {
		addrs, err := addressRecord(row.ID, byNode[row.ID])
		if err != nil {
			return nil, err
		}
		n, err := s.view(ctx, row, addrs)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

// Get is one Node, or ErrNotFound.
func (s *Service) Get(ctx context.Context, id ID) (Node, error) {
	q := db.New(s.cfg.Library)
	row, err := q.GetNode(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return Node{}, ErrNotFound
	}
	if err != nil {
		return Node{}, err
	}
	held, err := q.NodeAddresses(ctx, id)
	if err != nil {
		return Node{}, err
	}
	addrs, err := addressRecord(id, held)
	if err != nil {
		return Node{}, err
	}
	return s.view(ctx, row, addrs)
}

// addressRecord reads a Node's nodes_address rows through the peer
// address record's parser. No rows is the zero record.
func addressRecord(node ID, rows []string) (link.Addresses, error) {
	if len(rows) == 0 {
		return link.Addresses{}, nil
	}
	aps := make([]netip.AddrPort, len(rows))
	for i, a := range rows {
		ap, err := netip.ParseAddrPort(a)
		if err != nil {
			return link.Addresses{}, fmt.Errorf("node %s address %d: %w", node, i, err)
		}
		aps[i] = ap
	}
	addrs, err := link.ParseAddresses(aps)
	if err != nil {
		return link.Addresses{}, fmt.Errorf("node %s addresses: %w", node, err)
	}
	return addrs, nil
}

func (s *Service) view(ctx context.Context, row db.NodesNode, addrs link.Addresses) (Node, error) {
	key, err := link.KeyFromBytes(row.PublicKey)
	if err != nil {
		return Node{}, fmt.Errorf("node %s: %w", row.ID, err)
	}
	roles, err := s.assigned(ctx, s.cfg.Library, row.ID)
	if err != nil {
		return Node{}, err
	}
	version, err := s.controls.NodeVersion(ctx, row.ID)
	if err != nil {
		return Node{}, err
	}
	n := Node{
		ID:        row.ID,
		Name:      row.Name,
		Key:       key,
		Self:      row.ID == s.self,
		PairedAt:  row.PairedAt.Time(),
		Addresses: addrs,
		Roles:     roles,
		Version:   version,
	}
	n.Facts, _ = s.controls.NodeFacts(row.ID)
	n.Link, n.Since = s.Reach(row.ID)
	if n.Self {
		own := s.cfg.Node
		n.Build = &own
		if advertised, ok := s.cfg.Link.Advertised(); ok {
			n.Addresses = advertised
		}
		return n, nil
	}
	if up, ok := s.cfg.Link.State(key).(link.Live); ok {
		n.Build = &up.Peer
	}
	return n, nil
}
