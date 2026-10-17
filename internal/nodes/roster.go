package nodes

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// Timings are the roster's clocks.
type Timings struct {
	// SeenPersist is how often a Node's last Sync frame is written to
	// seen_at while it is in step. Its session's end writes it at once.
	SeenPersist time.Duration `yaml:"seen_persist"`
	// Blip is how long a Node that left reads Reconnecting.
	Blip time.Duration `yaml:"blip"`
	// Settle is how long a Node back from a lapse takes no new work.
	Settle time.Duration `yaml:"settle"`
}

func DefaultTimings() Timings {
	return Timings{SeenPersist: time.Minute, Blip: time.Minute, Settle: time.Minute}
}

var OfflineGrace = setting.Duration{
	Key:     "nodes.offlineGrace",
	Label:   "Offline grace",
	Default: 15 * time.Minute,
	Min:     time.Minute,
	Max:     24 * time.Hour,
	Step:    time.Minute,
}

func (s *Service) OfflineGraceField(ctx context.Context) (setting.DurationField, error) {
	grace, err := db.New(s.cfg.Library).GetOfflineGrace(ctx)
	if err != nil {
		return setting.DurationField{}, fmt.Errorf("read the offline grace: %w", err)
	}
	return OfflineGrace.Field(grace), nil
}

// SetOfflineGrace stores the offline grace and reads the roster again, so
// a Node away longer than the new grace lapses at once.
func (s *Service) SetOfflineGrace(ctx context.Context, v setting.Stored[int64]) error {
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).SetOfflineGrace(ctx, v.Override())
	})
	if err != nil {
		return fmt.Errorf("set the offline grace: %w", err)
	}
	return s.reload(ctx)
}

// Board is the roster now.
func (s *Service) Board() *Board { return s.board.Load() }

// Who answers who may hold r now.
func (s *Service) Who(r role.Name) Holders { return s.Board().Who(r) }

// Changed is closed at the next change of the board: an assignment, a
// session, a lapse or a settle.
func (s *Service) Changed() <-chan struct{} { return s.Board().Changed() }

// Serves reports whether node serves r now.
func (s *Service) Serves(r role.Name, node ID) bool { return s.Who(r).Serves(node) }

func (s *Service) Holds(r role.Name, node ID) bool { return s.Who(r).Holds(node) }

func (s *Service) Pick(r role.Name, prefer ID) (ID, bool) { return s.Who(r).Pick(prefer) }

func (s *Service) WrittenOff(r role.Name, node ID) bool { return s.Who(r).WrittenOff(node) }

// InStep reports whether node's Sync session is attached, always for the
// Controller.
func (s *Service) InStep(node ID) bool {
	v, ok := s.Board().Node(node)
	return ok && v.Standing == InStep
}

// Pause stops the grace clock until resume is called: no Node lapses
// while any pause is held.
func (s *Service) Pause() (resume func()) {
	s.update(func(in *inputs, now time.Time) { in.pause(now) })
	return func() { s.update(func(in *inputs, now time.Time) { in.resume(now) }) }
}

// update changes the roster's inputs, derives the board and wakes Run to
// find the board's next edge.
func (s *Service) update(change func(in *inputs, now time.Time)) {
	s.refresh(change)
	select {
	case s.rearm <- struct{}{}:
	default:
	}
}

// refresh changes the roster's inputs and derives the board. A board that
// reads differently replaces the last one, closing its Changed, and tells
// Admins about each Node it changed. A Node's node scope is touched when
// what it drains changed, and the Controller's when what it covers did.
func (s *Service) refresh(change func(in *inputs, now time.Time)) {
	s.rmu.Lock()
	now := time.Now()
	change(&s.in, now)
	next := derive(s.in, now)
	old := s.board.Load()
	swap := !next.same(old)
	if swap {
		s.board.Store(next)
	}
	s.rmu.Unlock()
	if !swap {
		return
	}
	close(old.changed)
	var touch []ID
	for _, id := range differing(old, next) {
		s.changed(id)
		if old.drains(id) != next.drains(id) {
			touch = append(touch, id)
		}
	}
	if old.covering() != next.covering() && !slices.Contains(touch, s.self) {
		touch = append(touch, s.self)
	}
	if len(touch) > 0 {
		s.controls.Touch(touch...)
	}
}

// differing is every Node whose view or Role states differ between a and b.
func differing(a, b *Board) []ID {
	var out []ID
	for i, v := range b.nodes {
		j, ok := a.index(v.ID)
		if !ok || a.nodes[j] != v || a.forfeit[j] != b.forfeit[i] || !maps.Equal(a.states[j], b.states[i]) {
			out = append(out, v.ID)
		}
	}
	for _, v := range a.nodes {
		if _, ok := b.index(v.ID); !ok {
			out = append(out, v.ID)
		}
	}
	return out
}

// reload reads the roster again, keeping what it remembers of each Node's
// sessions. Reloads run one at a time, so the last to run read the last
// commit.
func (s *Service) reload(ctx context.Context) error {
	s.reloading.Lock()
	defer s.reloading.Unlock()
	rows, grace, err := s.read(ctx)
	if err != nil {
		return err
	}
	s.update(func(in *inputs, _ time.Time) {
		in.grace = grace
		for i, n := range rows {
			if was := in.node(n.id); was != nil {
				rows[i].attached, rows[i].since, rows[i].seen = was.attached, was.since, was.seen
				rows[i].out, rows[i].mismatched, rows[i].back = was.out, was.mismatched, was.back
			}
		}
		in.nodes = rows
	})
	return nil
}

// read is the live rows, their Roles, forfeits and held work, and the
// grace.
func (s *Service) read(ctx context.Context) ([]nodeIn, time.Duration, error) {
	var (
		rows  []nodeIn
		grace time.Duration
	)
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if rows, err = loadRows(ctx, r); err != nil {
			return err
		}
		work, err := countWork(ctx, r, s.owners, rows)
		if err != nil {
			return err
		}
		for i := range rows {
			rows[i].work = work[rows[i].id]
		}
		override, err := db.New(r).GetOfflineGrace(ctx)
		grace = OfflineGrace.Value(override)
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("load the roster: %w", err)
	}
	return rows, grace, nil
}

// loadRows is every live row in id order with its assignment and forfeits.
func loadRows(ctx context.Context, r storage.Reader[file]) ([]nodeIn, error) {
	q := db.New(r)
	nodes, err := q.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := q.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	forfeits, err := q.ListForfeits(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]nodeIn, len(nodes))
	at := map[ID]int{}
	for i, n := range nodes {
		rows[i] = nodeIn{id: n.ID, name: n.Name, seen: n.SeenAt.Time()}
		at[n.ID] = i
	}
	for _, r := range roles {
		if i, ok := at[r.NodeID]; ok {
			rows[i].assigned = rows[i].assigned.With(r.Role)
		}
	}
	for _, f := range forfeits {
		if i, ok := at[f.NodeID]; ok {
			rows[i].forfeit = rows[i].forfeit.With(f.Role)
		}
	}
	return rows, nil
}

func (s *Service) inStep(node ID, attached bool) {
	s.update(func(in *inputs, now time.Time) {
		if attached {
			in.attach(node, now)
		} else {
			in.detach(node, now)
		}
	})
}

// stalled closes the link of a Node whose Sync stream went silent.
func (s *Service) stalled(node ID) {
	if key, ok := s.Key(node); ok {
		s.cfg.Link.Reset(key)
	}
}

func (s *Service) seen(node ID) {
	s.rmu.Lock()
	defer s.rmu.Unlock()
	if n := s.in.node(node); n != nil {
		n.seen = time.Now()
	}
}

// scope is the node scope: the Roles assigned to node, with the Roles the
// Controller covers on its own, then Always; the Roles node drains; and
// its name.
func (s *Service) scope(ctx context.Context, r storage.Reader[file], node ID) (role.NodeScope, error) {
	q := db.New(r)
	row, err := q.GetNode(ctx, node)
	if err != nil {
		return role.NodeScope{}, err
	}
	assigned, err := q.NodeRoles(ctx, node)
	if err != nil {
		return role.NodeScope{}, err
	}
	board := s.Board()
	holds := role.Of(assigned...)
	if node == s.self {
		holds |= board.covering()
	}
	return role.NodeScope{Holds: append(holds.Names(), s.cfg.Always...), Draining: board.drains(node).Names(), Name: row.Name}, nil
}

// touched notes that an owner's intent for nodes changed, so the work it
// counts may have: Run settles the forfeits and reads the roster again.
func (s *Service) touched(string, []ID) {
	select {
	case s.recount <- struct{}{}:
	default:
	}
}

// settle deletes each forfeit whose owner counts no work left on its Node,
// then reads the roster again, so the board's draining and written-off
// Roles follow what each owner still holds.
func (s *Service) settle(ctx context.Context) error {
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		forfeits, err := q.ListForfeits(ctx)
		if err != nil {
			return err
		}
		for _, f := range forfeits {
			held, err := s.owners[f.Role].Held(ctx, tx, f.NodeID)
			if err != nil {
				return fmt.Errorf("count %s work on node %s: %w", f.Role, f.NodeID, err)
			}
			if held.Units > 0 {
				continue
			}
			if err := q.DeleteForfeit(ctx, db.DeleteForfeitParams(f)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("settle the forfeits: %w", err)
	}
	return s.reload(ctx)
}

// situation is what Plan reads now: each live Node as a Seat, away unless
// board has it in step, with the work each owner counts on it.
func (s *Service) situation(ctx context.Context, r storage.Reader[file], board *Board) (Situation, error) {
	rows, err := loadRows(ctx, r)
	if err != nil {
		return Situation{}, err
	}
	seats := make([]Seat, len(rows))
	for i, n := range rows {
		v, ok := board.Node(n.id)
		held, moves, err := s.work(ctx, r, n.id)
		if err != nil {
			return Situation{}, err
		}
		seats[i] = Seat{
			Named:    Named{ID: n.id, Name: n.name},
			Assigned: n.assigned,
			Forfeit:  n.forfeit,
			Away:     !ok || v.Standing != InStep,
			Held:     held,
			Moves:    moves,
		}
	}
	return Situation{Controller: s.self, Seats: seats}, nil
}

// work is what each Role's owner counts on node: tied work as held, and
// portable work as moves.
func (s *Service) work(ctx context.Context, r storage.Reader[file], node ID) (held, moves map[role.Name]Held, err error) {
	held, moves = map[role.Name]Held{}, map[role.Name]Held{}
	for _, name := range role.Names() {
		h, err := s.owners[name].Held(ctx, r, node)
		if err != nil {
			return nil, nil, fmt.Errorf("count %s work on node %s: %w", name, node, err)
		}
		switch {
		case h.Units == 0:
		case name.Rule().Work == role.Tied:
			held[name] = h
		case name.Rule().Work == role.Portable:
			moves[name] = h
		}
	}
	return held, moves, nil
}

// Run follows the link and the board's clock until ctx ends: it notes
// which Nodes' links are live on another build, records the addresses a
// Worker's Hello advertises, re-derives the board whenever it wakes, so
// at each edge, settles the forfeits at its start and whenever an owner's
// intent changed, and writes seen_at every seen_persist and when a session
// ends.
func (s *Service) Run(ctx context.Context) error {
	changes := s.cfg.Link.Subscribe(ctx)
	s.touched("", nil)
	persist := time.NewTicker(s.cfg.Timings.SeenPersist)
	defer persist.Stop()
	edge := time.NewTimer(0)
	defer edge.Stop()
	defer s.persistSeen(context.WithoutCancel(ctx), true)
	for {
		s.refresh(func(*inputs, time.Time) {})
		s.rmu.Lock()
		next := nextEdge(s.in, time.Now())
		s.rmu.Unlock()
		edge.Stop()
		if !next.IsZero() {
			edge.Reset(time.Until(next))
		}
		select {
		case c, ok := <-changes:
			if !ok {
				return nil
			}
			s.follow(ctx, c)
		case <-edge.C:
		case <-s.rearm:
			s.persistSeen(ctx, false)
		case <-s.recount:
			if err := s.settle(ctx); err != nil && ctx.Err() == nil {
				s.cfg.Log.Error("settle what owners hold", "err", err.Error())
			}
		case <-persist.C:
			s.persistSeen(ctx, true)
		}
	}
}

func (s *Service) follow(ctx context.Context, c link.Change) {
	id, known := s.NodeOf(c.Key)
	if !known {
		return
	}
	up, live := c.State.(link.Live)
	s.update(func(in *inputs, _ time.Time) {
		if n := in.node(id); n != nil {
			n.mismatched = live && !up.Matched
		}
	})
	if live && !up.Peer.Addresses.IsZero() {
		if err := s.recordAddresses(ctx, id, up.Peer.Addresses.All()); err != nil && ctx.Err() == nil {
			s.cfg.Log.Error("record a Node's addresses", "node", id.String(), "err", err.Error())
		}
	}
	s.changed(id)
}

// persistSeen writes seen_at for every Node seen since it was last
// written, or only for those out of step unless all.
func (s *Service) persistSeen(ctx context.Context, all bool) {
	type seen struct {
		id ID
		at time.Time
	}
	var due []seen
	s.rmu.Lock()
	for _, n := range s.in.nodes {
		if (all || !n.attached) && n.seen.After(s.saved[n.id]) {
			due = append(due, seen{n.id, n.seen})
		}
	}
	s.rmu.Unlock()
	if len(due) == 0 {
		return
	}
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		for _, d := range due {
			if err := q.SetSeenAt(ctx, db.SetSeenAtParams{ID: d.id, SeenAt: storage.FromTime(d.at)}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if ctx.Err() == nil {
			s.cfg.Log.Error("write when Nodes were last seen", "err", err.Error())
		}
		return
	}
	s.rmu.Lock()
	for _, d := range due {
		s.saved[d.id] = d.at
	}
	s.rmu.Unlock()
}
