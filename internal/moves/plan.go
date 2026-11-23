package moves

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// WorkTo is one Work and the root it moves to.
type WorkTo struct {
	Work works.WorkID
	To   roots.ID
}

// RootTo is every Work of Root, moving to To.
type RootTo struct{ Root, To roots.ID }

// ClientTo is every torrent of the Download Client From, moving to To.
type ClientTo struct{ From, To Side }

type Options struct {
	// IncludeLinkedTorrents moves the torrent a Work's files are linked
	// to with the Work.
	IncludeLinkedTorrents bool
	// RepointRouting has each Tracker route new Grabs to the
	// destination client of its torrents.
	RepointRouting     bool
	DeleteEmptiedRoots bool
}

// Ask is what an Admin previews.
type Ask struct {
	Origin   Origin
	Works    []WorkTo
	Roots    []RootTo
	Torrents []TorrentTo
	Clients  []ClientTo
	Options  Options
}

// Token names one preview, which a Start takes.
type Token string

func (t Token) hash() []byte {
	h := sha256.Sum256([]byte(t))
	return h[:]
}

// Plan is one preview: a unit and its verdict per Work with files, and the
// Works without files a root origin homes.
type Plan struct {
	Token     Token
	Ask       Ask
	Units     []PlanUnit
	Homed     []WorkTo
	ExpiresAt time.Time
	// Recommendation is the plan's sentence and Button its Start's label,
	// such as "Move 41 of 43".
	Recommendation, Button string
}

// PlanUnit is one planned unit. Bytes are what copies; a link copies none.
// Torrent is a torrent unit's, Subject zero.
type PlanUnit struct {
	Subject WorkSubject
	Torrent *TorrentSubject
	// Linked is the torrent a Work unit places its linked files from,
	// whose unit the plan puts before it.
	Linked  *downloadclients.ID
	Label   string
	Lane    Lane
	Files   []File
	Bytes   int64
	Verdict Verdict
}

// Movable is how many of p's units aren't red.
func (p Plan) Movable() int {
	n := 0
	for _, u := range p.Units {
		if u.Verdict.Color != Red {
			n++
		}
	}
	return n
}

// plans keeps the latest previews, the oldest dropped past n.
type plans struct {
	mu    sync.Mutex
	n     int
	order []Token
	of    map[Token]Plan
}

func newPlans(n int) *plans { return &plans{n: n, of: map[Token]Plan{}} }

func (p *plans) put(plan Plan) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.of[plan.Token] = plan
	p.order = append(p.order, plan.Token)
	for len(p.order) > p.n {
		delete(p.of, p.order[0])
		p.order = p.order[1:]
	}
}

func (p *plans) get(t Token, now time.Time) (Plan, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	plan, ok := p.of[t]
	return plan, ok && now.Before(plan.ExpiresAt)
}

func newToken() Token {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return Token(base64.RawURLEncoding.EncodeToString(b))
}

// Preview plans ask: it reads the library in pages of plan_batch Works,
// probes each source Node in pages, and judges every unit.
func (m *Moves) Preview(ctx context.Context, ask Ask) (Plan, error) {
	subjects, homed, err := m.resolve(ctx, ask)
	if err != nil {
		return Plan{}, err
	}
	var units []PlanUnit
	var facts []planFacts
	for page := range slices.Chunk(subjects, m.cfg.Timings.PlanBatch) {
		us, fs, err := m.planPage(ctx, page)
		if err != nil {
			return Plan{}, err
		}
		units, facts = append(units, us...), append(facts, fs...)
	}
	if err := m.rooms(ctx, units, facts); err != nil {
		return Plan{}, err
	}
	for i := range units {
		units[i].Verdict = verdictOf(units[i], facts[i])
	}
	if ask.Options.IncludeLinkedTorrents {
		if units, err = m.linkTorrents(ctx, units); err != nil {
			return Plan{}, err
		}
	}
	ts, err := m.clientTorrents(ctx, ask)
	if err != nil {
		return Plan{}, err
	}
	torrents, err := m.planTorrents(ctx, ts, ask.Origin == OriginDrain)
	if err != nil {
		return Plan{}, err
	}
	units = append(units, torrents...)
	var named map[role.NodeID]string
	err = m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		named, err = nodeNamesIn(ctx, r)
		return err
	})
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{Token: newToken(), Ask: ask, Units: units, Homed: homed, ExpiresAt: m.now().Add(m.cfg.Timings.PlanLifetime)}
	m.recommend(&plan, named)
	m.plans.put(plan)
	return plan, nil
}

// resolve is ask's Works, a root's read in pages, and the Works of a
// root without files, which a Start homes.
func (m *Moves) resolve(ctx context.Context, ask Ask) (subjects, homed []WorkTo, err error) {
	subjects = slices.Clone(ask.Works)
	if err := m.rootsKnown(ctx, ask.Roots); err != nil {
		return nil, nil, err
	}
	for _, rt := range ask.Roots {
		var after works.WorkID
		for {
			var page []roots.Homed
			err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
				var err error
				page, err = m.cfg.Roots.WorksIn(ctx, r, rt.Root, after, m.cfg.Timings.PlanBatch)
				return err
			})
			if err != nil {
				return nil, nil, err
			}
			for _, h := range page {
				if h.HasFiles {
					subjects = append(subjects, WorkTo{Work: h.Work, To: rt.To})
				} else {
					homed = append(homed, WorkTo{Work: h.Work, To: rt.To})
				}
			}
			if len(page) < m.cfg.Timings.PlanBatch {
				break
			}
			after = page[len(page)-1].Work
		}
	}
	return subjects, homed, nil
}

// rootsKnown is ErrNoRoot unless every root of rs is one.
func (m *Moves) rootsKnown(ctx context.Context, rs []RootTo) error {
	var ids []roots.ID
	for _, rt := range rs {
		ids = appendOnce(appendOnce(ids, rt.Root), rt.To)
	}
	if len(ids) == 0 {
		return nil
	}
	return m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		dests, err := m.cfg.Destinations(ctx, r, ids)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if _, ok := dests[id]; !ok {
				return ErrNoRoot
			}
		}
		return nil
	})
}

// clientTorrents is ask's torrents, then every torrent of each of its
// clients.
func (m *Moves) clientTorrents(ctx context.Context, ask Ask) ([]TorrentTo, error) {
	out := slices.Clone(ask.Torrents)
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		for _, c := range ask.Clients {
			ids, err := downloadclients.TorrentsIn(ctx, r, c.From.Node, c.From.Client)
			if err != nil {
				return err
			}
			for _, id := range ids {
				out = append(out, TorrentTo{Torrent: id, To: c.To})
			}
		}
		return nil
	})
	return out, err
}

// pageRead is what one page's Read answers.
type pageRead struct {
	summaries map[works.WorkID]works.Summary
	homes     map[works.WorkID]roots.ID
	sits      map[files.ID]files.Sitting
	moving    map[works.WorkID]bool
	dests     map[roots.ID]roots.Destination
	covers    []mediaservers.Cover
	// plays is whether a Media Server plays each file now.
	plays map[files.ID]bool
	// budgets is each root's path budget.
	budgets map[roots.ID]int
}

func (m *Moves) planPage(ctx context.Context, page []WorkTo) ([]PlanUnit, []planFacts, error) {
	ws := make([]works.WorkID, len(page))
	for i, s := range page {
		ws[i] = s.Work
	}
	var rd pageRead
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		rd, err = m.readPage(ctx, r, page, ws)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	for _, s := range page {
		if _, ok := rd.summaries[s.Work]; !ok {
			return nil, nil, works.ErrNoWork
		}
		if _, ok := rd.dests[s.To]; !ok {
			return nil, nil, ErrNoRoot
		}
	}
	byWork := map[works.WorkID][]files.ID{}
	for id, s := range rd.sits {
		byWork[s.Work] = append(byWork[s.Work], id)
	}
	for _, ids := range byWork {
		slices.SortFunc(ids, func(a, b files.ID) int { return a.Compare(b.UUID) })
	}
	probes := m.probeAll(ctx, page, rd, byWork)
	units := make([]PlanUnit, 0, len(page))
	facts := make([]planFacts, 0, len(page))
	for _, s := range page {
		u, f := m.planUnit(s, rd, byWork[s.Work], probes[s.Work])
		units, facts = append(units, u), append(facts, f)
	}
	return units, facts, nil
}

// readPage is what the planner reads in r for page's Works ws.
func (m *Moves) readPage(ctx context.Context, r storage.Reader[file], page []WorkTo, ws []works.WorkID) (pageRead, error) {
	var rd pageRead
	var err error
	if rd.summaries, err = works.SummariesOf(ctx, r, ws); err != nil {
		return rd, err
	}
	if rd.homes, err = roots.HomesOf(ctx, r, works.Only(ws)); err != nil {
		return rd, err
	}
	if rd.sits, err = files.Sits(ctx, r, ws); err != nil {
		return rd, err
	}
	if rd.moving, err = movingOf(ctx, r, ws); err != nil {
		return rd, err
	}
	if rd.budgets, err = roots.BudgetsIn(ctx, r); err != nil {
		return rd, err
	}
	var ids []roots.ID
	for _, s := range page {
		ids = appendOnce(ids, s.To)
		if from, ok := rd.homes[s.Work]; ok {
			ids = appendOnce(ids, from)
		}
	}
	if rd.dests, err = m.cfg.Destinations(ctx, r, ids); err != nil {
		return rd, err
	}
	if rd.covers, err = m.cfg.MediaServers.Coverage(ctx, r, ids); err != nil {
		return rd, err
	}
	fs := make([]files.ID, 0, len(rd.sits))
	for id := range rd.sits {
		fs = append(fs, id)
	}
	on, err := m.cfg.MediaServers.Playing(ctx, r, fs)
	rd.plays = map[files.ID]bool{}
	for id, servers := range on {
		rd.plays[id] = len(servers) > 0
	}
	return rd, err
}

// movingOf is which of ws an open unit moves, read in r.
func movingOf(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[works.WorkID]bool, error) {
	open, err := db.New(r).ListOpenUnitsOfWorks(ctx, ws)
	out := make(map[works.WorkID]bool, len(open))
	for _, u := range open {
		out[u.WorkID] = true
	}
	return out, err
}

func appendOnce[T comparable](s []T, v T) []T {
	if slices.Contains(s, v) {
		return s
	}
	return append(s, v)
}

// probed is one Work's share of its Nodes' probes: the source Node's
// page, and the destination Node's when it is another.
type probed struct {
	src, dst storing.Probed
	err      error
}

// probeAsk is one Work's part of a probe page: its files as sources, as
// destinations, or both on one Node.
type probeAsk struct {
	work          works.WorkID
	from, to      roots.ID
	rels          []storing.Rel
	sources, dest bool
}

// probePage asks node about page.
func (m *Moves) probePage(ctx context.Context, node role.NodeID, page []probeAsk) (storing.Probed, error) {
	req := storing.ProbeMove{}
	for _, a := range page {
		if a.sources {
			req.Sources = addRels(req.Sources, a.from, a.rels)
		}
		if a.dest {
			req.Destinations = addRels(req.Destinations, a.to, a.rels)
		}
	}
	ctx, cancel := context.WithTimeout(ctx, m.cfg.Timings.ProbeDeadline)
	defer cancel()
	return m.cfg.Disk.ProbeMove(ctx, node, req)
}

// probeAll probes the page's Works on their Nodes: each source file with
// its companions on its source Node, and each destination path's
// occupants on the destination's Node.
func (m *Moves) probeAll(ctx context.Context, page []WorkTo, rd pageRead, byWork map[works.WorkID][]files.ID) map[works.WorkID]probed {
	out := map[works.WorkID]probed{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, m.cfg.Timings.ProbesInFlight)
	for node, asks := range asksOf(page, rd, byWork) {
		for _, pg := range pages(asks, func(a probeAsk) []storing.Rel { return a.rels }, m.cfg.Timings.ProbePage, m.cfg.Timings.ProbePageBytes) {
			wg.Go(func() {
				slots <- struct{}{}
				defer func() { <-slots }()
				got, err := m.probePage(ctx, node, pg)
				mu.Lock()
				defer mu.Unlock()
				for _, a := range pg {
					p := out[a.work]
					if a.sources {
						p.src = got
					} else {
						p.dst = got
					}
					p.err = cmp.Or(p.err, err)
					out[a.work] = p
				}
			})
		}
	}
	wg.Wait()
	return out
}

// asksOf is each Node's share of the page's probes: the sources, with the
// destinations on the same Node, and the destinations on another.
func asksOf(page []WorkTo, rd pageRead, byWork map[works.WorkID][]files.ID) map[role.NodeID][]probeAsk {
	byNode := map[role.NodeID][]probeAsk{}
	for _, s := range page {
		from, ok := rd.homes[s.Work]
		src, known := rd.dests[from]
		if !ok || !known {
			continue
		}
		a := probeAsk{work: s.Work, from: from, to: s.To, sources: true}
		for _, id := range byWork[s.Work] {
			a.rels = append(a.rels, rd.sits[id].Sits)
		}
		dst, known := rd.dests[s.To]
		a.dest = known && dst.Node == src.Node
		byNode[src.Node] = append(byNode[src.Node], a)
		if known && !a.dest {
			byNode[dst.Node] = append(byNode[dst.Node], probeAsk{work: s.Work, from: from, to: s.To, rels: a.rels, dest: true})
		}
	}
	return byNode
}

// pages splits asks into pages of at most probe_page paths and
// probe_page_bytes of path bytes, one ask never split.
func pages[T any](asks []T, rels func(T) []storing.Rel, most, bytes int) [][]T {
	var out [][]T
	var cur []T
	n, size := 0, 0
	for _, a := range asks {
		rs := rels(a)
		b := 0
		for _, r := range rs {
			b += 2 * len(r.String())
		}
		if len(cur) > 0 && (n+len(rs) > most || size+b > bytes) {
			out, cur, n, size = append(out, cur), nil, 0, 0
		}
		cur, n, size = append(cur, a), n+len(rs), size+b
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out
}

func addRels(rs []storing.ProbeRoot, root roots.ID, rels []storing.Rel) []storing.ProbeRoot {
	i := slices.IndexFunc(rs, func(r storing.ProbeRoot) bool { return r.Root == root.UUID })
	if i < 0 {
		return append(rs, storing.ProbeRoot{Root: root.UUID, Rels: slices.Clone(rels)})
	}
	rs[i].Rels = append(rs[i].Rels, rels...)
	return rs
}

func (m *Moves) planUnit(s WorkTo, rd pageRead, ids []files.ID, p probed) (PlanUnit, planFacts) {
	sum := rd.summaries[s.Work]
	from := rd.dests[rd.homes[s.Work]]
	to := rd.dests[s.To]
	u := PlanUnit{
		Subject: WorkSubject{Work: s.Work, Type: sum.ContentType, From: End{Root: from.ID, Node: from.Node}, To: End{Root: to.ID, Node: to.Node}},
		Label:   sum.Label,
	}
	f := planFacts{from: from, to: to, moving: rd.moving[s.Work], probe: p.err, covers: rd.covers, budget: rd.budgets[s.To]}
	if from.Volume != nil && to.Volume != nil {
		u.Lane = laneOf(u.Subject.From, u.Subject.To, from.Volume.ID, to.Volume.ID)
	}
	var rels []storing.Rel
	for _, id := range ids {
		sat := rd.sits[id]
		f.playing = f.playing || rd.plays[id]
		f.longest = max(f.longest, len(sat.Path.String()))
		at := storing.Library{Root: from.ID.UUID, Rel: sat.Sits}
		got, ok := p.src.Files[at]
		if !ok {
			f.missing++
			continue
		}
		rels = append(rels, sat.Sits)
		u.Files = append(u.Files, File{Position: len(u.Files), Kind: KindLibrary, ID: id, Sits: sat.Sits, Size: got.Size, Seen: got.Seen})
		for _, c := range p.src.Companions[at] {
			kind := KindCompanion
			if c.Ledger {
				kind = KindSidecar
			}
			u.Files = append(u.Files, File{Position: len(u.Files), Kind: kind, Sits: c.Rel, Size: c.Size, Seen: c.Seen, Ledger: c.Ledger})
		}
	}
	for _, file := range u.Files {
		f.longest = max(f.longest, len(file.Sits.String()))
	}
	dst := p.src
	if to.Node != from.Node {
		dst = p.dst
	}
	if p.err == nil {
		f.occupied = occupants(dst, to.ID, rels)
	}
	if !sameVolume(from, to) {
		for _, file := range u.Files {
			u.Bytes += file.Size
		}
	}
	return u, f
}

// rooms gives each unit that copies the room left on its destination
// volume once the units before it took theirs.
func (m *Moves) rooms(ctx context.Context, units []PlanUnit, facts []planFacts) error {
	room := map[volumeOf]*int64{}
	return m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		for i, u := range units {
			if u.Bytes == 0 || facts[i].to.Volume == nil {
				continue
			}
			left, err := m.roomOn(ctx, r, room, facts[i].to)
			if err != nil {
				return err
			}
			if left == nil {
				continue
			}
			facts[i].room = new(*left)
			if facts[i].probe == nil && *left >= u.Bytes {
				*left -= u.Bytes
			}
		}
		return nil
	})
}

type volumeOf struct {
	node role.NodeID
	id   string
}

// roomOn is the bytes left on to's volume, read once per volume into
// room, nil when its room is unknown.
func (m *Moves) roomOn(ctx context.Context, r storage.Reader[file], room map[volumeOf]*int64, to roots.Destination) (*int64, error) {
	v := volumeOf{to.Node, to.Volume.ID}
	if left, ok := room[v]; ok {
		return left, nil
	}
	left, known, err := m.cfg.Imports.CopyRoom(ctx, r, to.ID)
	if err != nil {
		return nil, err
	}
	if known {
		room[v] = &left
	} else {
		room[v] = nil
	}
	return room[v], nil
}
