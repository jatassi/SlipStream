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
	"time"

	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes/internal/db"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
	"github.com/jatassi/slipstream/internal/role"
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
	// Served is every Role's control half.
	Served []role.Served
	// Always is Roles every Node holds beyond its nodes_role rows: only a
	// test's, as the simulation tier runs roletest's Probe.
	Always []role.Name
	Log    *slog.Logger
}

var (
	ErrNotFound      = errors.New("no Node has this id")
	ErrNoLinkAddress = errors.New("the link has no address to advertise")
	ErrLifetime      = errors.New("the Join Code lifetime is 10 minutes to 7 days")
)

const (
	MinJoinCodeLifetime = 10 * time.Minute
	MaxJoinCodeLifetime = 7 * 24 * time.Hour
)

// Service is the Controller's roster and its Nodes' records. Its memory
// holds what calls read on every request: each Node's key and Roles.
type Service struct {
	cfg      Config
	self     ID
	name     string
	opened   time.Time
	controls *role.Controls

	mu      sync.Mutex
	members map[ID]member
	byKey   map[link.Key]ID
	since   map[link.Key]time.Time
	wake    map[ID]chan struct{}
}

type member struct {
	key   link.Key
	roles []role.Name
}

// Open creates this Node's own row when it has none, holding every Role
// but Relaying, loads the roster and builds the control half.
func Open(ctx context.Context, cfg Config) (*Service, error) {
	s := &Service{
		cfg:     cfg,
		opened:  time.Now(),
		members: map[ID]member{},
		byKey:   map[link.Key]ID{},
		since:   map[link.Key]time.Time{},
		wake:    map[ID]chan struct{}{},
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
	s.controls = role.NewControls(role.ControlsConfig{
		Library:  cfg.Library,
		Nodes:    s,
		Assigned: s.Assigned,
		Changed:  s.changed,
		Log:      cfg.Log,
	}, cfg.Served...)
	return s, nil
}

func (s *Service) load(ctx context.Context) error {
	q := db.New(s.cfg.Library)
	rows, err := q.ListNodes(ctx)
	if err != nil {
		return fmt.Errorf("load the roster: %w", err)
	}
	roles, err := q.ListRoles(ctx)
	if err != nil {
		return fmt.Errorf("load the roster: %w", err)
	}
	held := map[ID][]role.Name{}
	for _, r := range roles {
		held[r.NodeID] = append(held[r.NodeID], r.Role)
	}
	for _, row := range rows {
		key, err := link.KeyFromBytes(row.PublicKey)
		if err != nil {
			return fmt.Errorf("load the roster: node %s: %w", row.ID, err)
		}
		s.members[row.ID] = member{key: key, roles: ordered(held[row.ID])}
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
	m, ok := s.members[node]
	return m.key, ok
}

// Holders is every Node holding r, in id order.
func (s *Service) Holders(r role.Name) []ID {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ids []ID
	for id, m := range s.members {
		if slices.Contains(m.roles, r) {
			ids = append(ids, id)
		}
	}
	slices.SortFunc(ids, func(a, b ID) int { return a.Compare(b.UUID) })
	return ids
}

// HoldersByPairing is every Node holding r, first paired first.
func (s *Service) HoldersByPairing(ctx context.Context, r role.Name) ([]ID, error) {
	return db.New(s.cfg.Library).HoldersByPairing(ctx, r)
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

// Assigned derives the node scope: the Roles assigned to node, in
// role.Names() order, then Always.
func (s *Service) Assigned(ctx context.Context, r storage.Reader[file], node ID) ([]role.Name, error) {
	roles, err := db.New(r).NodeRoles(ctx, node)
	if err != nil {
		return nil, err
	}
	return append(ordered(roles), s.cfg.Always...), nil
}

func ordered(roles []role.Name) []role.Name {
	names := role.Names()
	out := slices.Clone(roles)
	slices.SortFunc(out, func(a, b role.Name) int { return slices.Index(names, a) - slices.Index(names, b) })
	return out
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
	if _, ok := s.members[row.ID]; !ok {
		s.members[row.ID] = member{key: peer}
		s.byKey[peer] = row.ID
	}
	s.mu.Unlock()
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
			err = q.InsertNode(ctx, db.InsertNodeParams(row))
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
		seconds, err := q.GetJoinCodeLifetime(ctx)
		if err != nil {
			return err
		}
		now := storage.Now()
		expires = storage.FromTime(now.Time().Add(time.Duration(seconds) * time.Second))
		return q.InsertJoinCode(ctx, db.InsertJoinCodeParams{SecretHash: hash[:], CreatedAt: now, ExpiresAt: expires})
	})
	if err != nil {
		return JoinCode{}, fmt.Errorf("mint a Join Code: %w", err)
	}
	return JoinCode{Code: code, ExpiresAt: expires.Time()}, nil
}

// JoinCodeLifetime is how long a new Join Code is accepted.
func (s *Service) JoinCodeLifetime(ctx context.Context) (time.Duration, error) {
	seconds, err := db.New(s.cfg.Library).GetJoinCodeLifetime(ctx)
	return time.Duration(seconds) * time.Second, err
}

func (s *Service) SetJoinCodeLifetime(ctx context.Context, d time.Duration) error {
	if d < MinJoinCodeLifetime || d > MaxJoinCodeLifetime || d%time.Second != 0 {
		return ErrLifetime
	}
	return s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).SetJoinCodeLifetime(ctx, int64(d/time.Second))
	})
}

// SetRoles makes roles the set node holds and tells the control half. For
// a Node in step, here or over a live link, it returns once the Node's
// facts show the node-scope version derived from that set, or ctx ends.
func (s *Service) SetRoles(ctx context.Context, node ID, roles []role.Name) (Node, error) {
	want := ordered(slices.Compact(slices.Sorted(slices.Values(roles))))
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return replaceRoles(ctx, db.New(tx), node, want)
	})
	if err != nil {
		return Node{}, err
	}
	s.mu.Lock()
	m := s.members[node]
	m.roles = want
	s.members[node] = m
	s.mu.Unlock()
	s.controls.Touch(node)
	s.changed(node)
	if err := s.awaitDelivery(ctx, node); err != nil {
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

// awaitDelivery waits while node is in step and its facts lag the node
// scope the library derives now.
func (s *Service) awaitDelivery(ctx context.Context, node ID) error {
	for {
		woken := s.awaitChange(node)
		if !s.inStep(node) {
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

// AwaitInStep returns once node's link is live and matched, at once for
// this Node, or with ctx's error when it ends first.
func (s *Service) AwaitInStep(ctx context.Context, node ID) error {
	for {
		woken := s.awaitChange(node)
		if s.inStep(node) {
			return nil
		}
		select {
		case <-woken:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Service) inStep(node ID) bool {
	if node == s.self {
		return true
	}
	key, ok := s.Key(node)
	if !ok {
		return false
	}
	up, ok := s.cfg.Link.State(key).(link.Live)
	return ok && up.Matched
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

// changed wakes whoever waits on node and tells Admins. It never blocks.
func (s *Service) changed(node ID) {
	s.mu.Lock()
	ch := s.wake[node]
	delete(s.wake, node)
	s.mu.Unlock()
	if ch != nil {
		close(ch)
	}
	s.cfg.Changed.Publish(context.Background(), Changed{NodeID: node.String()})
}

// Run follows the link until ctx ends: when a Node's session starts or
// ends it notes the time, records the addresses a Worker's Hello
// advertises, and publishes nodes.changed.
func (s *Service) Run(ctx context.Context) error {
	for c := range s.cfg.Link.Subscribe(ctx) {
		s.mu.Lock()
		s.since[c.Key] = time.Now()
		id, known := s.byKey[c.Key]
		s.mu.Unlock()
		if !known {
			continue
		}
		if up, ok := c.State.(link.Live); ok && !up.Peer.Addresses.IsZero() {
			if err := s.recordAddresses(ctx, id, up.Peer.Addresses.All()); err != nil && ctx.Err() == nil {
				s.cfg.Log.Error("record a Node's addresses", "node", id.String(), "err", err.Error())
			}
		}
		s.changed(id)
	}
	return nil
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
	roles, err := s.Assigned(ctx, s.cfg.Library, row.ID)
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
		Since:     s.opened,
	}
	n.Facts, _ = s.controls.NodeFacts(row.ID)
	if n.Self {
		own := s.cfg.Node
		n.Link, n.Build = Local, &own
		if advertised, ok := s.cfg.Link.Advertised(); ok {
			n.Addresses = advertised
		}
		return n, nil
	}
	s.mu.Lock()
	if since, ok := s.since[key]; ok {
		n.Since = since
	}
	s.mu.Unlock()
	if up, ok := s.cfg.Link.State(key).(link.Live); ok {
		n.Link, n.Build = Up, &up.Peer
		if !up.Matched {
			n.Link = Mismatched
		}
	}
	return n, nil
}
