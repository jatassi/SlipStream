package transfers

import (
	"bytes"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/nodes"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// judgment is judge's whole answer.
type judgment struct {
	// views holds every Transfer, in order: (Created, ID).
	views   []View
	intents intents
	// fail is the failures judge decides from the snapshot. The loop
	// stores them.
	fail []judged
	// next is the earliest instant after now at which the answer changes
	// with no other input changing, zero for never.
	next time.Time
}

type judged struct {
	id       ID
	code     Code
	position int
}

func (j *judgment) view(id ID) (View, bool) {
	i := slices.IndexFunc(j.views, func(v View) bool { return v.ID == id })
	if i < 0 {
		return View{}, false
	}
	return j.views[i], true
}

// intents is every paired Node's intent.
type intents map[role.NodeID]*tv1.Intent

func (in intents) of(node role.NodeID) *tv1.Intent {
	if i, ok := in[node]; ok {
		return i
	}
	return &tv1.Intent{}
}

// line is the Transfers that may copy on one pair, in order.
type line struct {
	key   pairKey
	cands []candidate
}

type candidate struct {
	i int
	t Transfer
	// files is what it sends now: its missing files that have room.
	files []File
	d     dest
	recv  *tv1.Receiving
	// aside is the wait its receiver reported that steps it aside.
	aside *Wait
}

// judge is the one rule for every live state, intent and wait. It does no
// IO and reads no clock but s.now.
func judge(s snapshot) judgment {
	j := judgment{views: make([]View, len(s.transfers)), intents: intents{}}
	for node := range s.peers {
		j.intent(s, node)
	}
	var lines []*line
	reserved := map[volumeAt]int64{}
	for i, t := range s.transfers {
		d := s.dest(t)
		recv := s.receiving(t, d)
		in := j.intent(s, d.node)
		in.Keep = append(in.Keep, kept(t))
		j.intent(s, t.Source)
		v := s.base(t, d, recv)
		if t.Failure == nil {
			if code, position, ok := s.judgedFailure(t, d); ok {
				j.fail = append(j.fail, judged{t.ID, code, position})
				v.Failure = &Failure{Code: code, Position: position}
			}
		}
		switch {
		case v.Failure != nil:
			v.State = Failed
		case !slices.ContainsFunc(t.Files, missing):
			v.State, v.Verified = Copying, v.Total
		default:
			if w := s.stepAside(t, d); w != nil {
				v.State, v.Wait = Waiting, w
			} else if files, sp := s.space(t, d, recv, reserved); sp != nil {
				v.State, v.Space = NeedsSpace, sp
			} else {
				key := pairKey{t.Source, d.node}
				at := slices.IndexFunc(lines, func(l *line) bool { return l.key == key })
				if at < 0 {
					lines, at = append(lines, &line{key: key}), len(lines)
				}
				lines[at].cands = append(lines[at].cands, candidate{i: i, t: t, files: files, d: d, recv: recv, aside: s.reported(t, d, recv)})
			}
		}
		j.views[i] = v
	}
	for _, l := range lines {
		j.run(s, l)
	}
	for _, in := range j.intents {
		slices.SortFunc(in.Keep, func(a, b *tv1.Kept) int { return bytes.Compare(a.GetTransferId(), b.GetTransferId()) })
	}
	return j
}

func (j *judgment) intent(s snapshot, node role.NodeID) *tv1.Intent {
	in, ok := j.intents[node]
	if !ok {
		in = &tv1.Intent{SendLimit: s.limits[node]}
		j.intents[node] = in
	}
	return in
}

// dest is where a Transfer's files go now, as judge reads it. root is
// nil for the two Backup kinds, a download folder and a scratch file,
// which have no root State and no space step: a full disk fails a
// Backup's copy at the receiver, and makes the others' wait.
type dest struct {
	node  role.NodeID
	place int64
	root  *roots.Destination
}

// dest is t's destination in s.
func (s snapshot) dest(t Transfer) dest {
	switch d := t.Destination.(type) {
	case Root:
		r := s.dests[d.ID]
		return dest{node: r.Node, place: r.Place, root: &r}
	case BackupDestination:
		return dest{node: d.Node, place: backupPlace}
	case BackupsFolder:
		return dest{node: d.Node, place: backupPlace}
	case DownloadFolder:
		return dest{node: d.Node, place: backupPlace}
	case Scratch:
		return dest{node: d.Node, place: backupPlace}
	}
	panic("transfers: a destination of no known kind")
}

// wire is t's destination as its receiver's intent names it. A Backup
// kind names its one file's Backup name, which the receiver renames the
// file to once it verifies.
func (d dest) wire(t Transfer) *tv1.Destination {
	switch dd := t.Destination.(type) {
	case Root:
		return &tv1.Destination{Kind: &tv1.Destination_Root{Root: &tv1.RootDestination{RootId: dd.ID.Bytes(), Place: d.place}}}
	case BackupDestination:
		return &tv1.Destination{Kind: &tv1.Destination_Backup{Backup: &tv1.BackupPlace{Folder: dd.Folder, Name: t.Files[0].Name}}}
	case BackupsFolder:
		return &tv1.Destination{Kind: &tv1.Destination_BackupsFolder{BackupsFolder: &tv1.BackupPlace{Folder: dd.Folder, Name: t.Files[0].Name}}}
	case DownloadFolder:
		w := &tv1.DownloadFolder{ContentType: dd.ContentType, InfoHash: dd.InfoHash[:]}
		if dd.Client != nil {
			w.ClientId = dd.Client[:]
		}
		return &tv1.Destination{Kind: &tv1.Destination_DownloadFolder{DownloadFolder: w}}
	case Scratch:
		return &tv1.Destination{Kind: &tv1.Destination_Scratch{Scratch: dd.Name}}
	}
	panic("transfers: a destination of no known kind")
}

// missing is a file its receiver still has to copy.
func missing(f File) bool { return !f.Verified && !f.Placed }

// kept is t's staging as its receiver keeps it: every position its
// purpose still wants.
func kept(t Transfer) *tv1.Kept {
	k := &tv1.Kept{TransferId: t.ID.Bytes()}
	for _, f := range t.Files {
		k.Positions = append(k.Positions, uint32(f.Position)) //nolint:gosec // G115: a position counts a Request's files
	}
	return k
}

// base is t's view before judge decides its state: its bytes, its
// receiver's details and its pair's last speed.
func (s snapshot) base(t Transfer, d dest, recv *tv1.Receiving) View {
	v := View{Transfer: t, DestNode: d.node, Rate: s.pairs[pairKey{t.Source, d.node}].Rate}
	for _, f := range t.Files {
		v.Total += f.Size
		if !missing(f) {
			v.Verified += f.Size
		}
	}
	if recv != nil {
		v.Verified = min(v.Total, v.Verified+recv.GetVerifiedBytes())
		v.Details = Details{Lanes: int(recv.GetLanes()), RTTMillis: int(recv.GetRttMs()), Retried: int(recv.GetRetried())}
	}
	return v
}

// receiving is t's facts from its destination Node, nil while it has
// none.
func (s snapshot) receiving(t Transfer, d dest) *tv1.Receiving {
	for _, r := range s.facts[d.node].GetReceiving() {
		if bytes.Equal(r.GetTransferId(), t.ID.Bytes()) {
			return r
		}
	}
	return nil
}

func (s snapshot) sending(t Transfer) *tv1.Sending {
	for _, r := range s.facts[t.Source].GetSending() {
		if bytes.Equal(r.GetTransferId(), t.ID.Bytes()) {
			return r
		}
	}
	return nil
}

// judgedFailure is the failure the snapshot alone decides: the root moved
// onto the source Node, or the source's Node was written off with a file
// still to copy.
func (s snapshot) judgedFailure(t Transfer, d dest) (Code, int, bool) {
	if d.node == t.Source {
		return Rerouted, -1, true
	}
	for _, f := range t.Files {
		if missing(f) && s.sources[sourceAt{t.Source, f.Source}].State == source.WrittenOff {
			return SourceWrittenOff, f.Position, true
		}
	}
	return "", 0, false
}

// stepAside is the wait the Controller's own inputs give t alone.
func (s snapshot) stepAside(t Transfer, d dest) *Wait {
	if r := d.root; r != nil {
		if s.ends[d.node].DrainsStorage {
			return &Wait{Kind: StorageDraining, Node: d.node}
		}
		if r.State != roots.Ready && r.State != roots.Offline {
			return &Wait{Kind: RootNotReady, Node: d.node, Root: r.ID, Detail: r.Detail}
		}
	}
	for _, f := range t.Files {
		if !missing(f) {
			continue
		}
		if r := s.sources[sourceAt{t.Source, f.Source}]; r.State == source.Checking || r.State == source.Incomplete {
			return &Wait{Kind: SourceChecking, Node: t.Source, Name: r.Name}
		}
	}
	return nil
}

// reported is the step-aside wait t's receiver last reported, nil for
// none.
func (s snapshot) reported(t Transfer, d dest, recv *tv1.Receiving) *Wait {
	w := recv.GetWait()
	switch w.GetKind() {
	case tv1.Wait_KIND_SOURCE_NOT_READY:
		f := t.Files[slices.IndexFunc(t.Files, missing)]
		return &Wait{Kind: SourceNotReady, Node: t.Source, Name: f.Name, Detail: w.GetDetail()}
	case tv1.Wait_KIND_ROOT_NOT_READY:
		if d.root == nil {
			return &Wait{Kind: FolderNotReady, Node: d.node, Detail: w.GetDetail()}
		}
		return &Wait{Kind: RootNotReady, Node: d.node, Root: d.root.ID, Detail: w.GetDetail()}
	case tv1.Wait_KIND_UNSPECIFIED, tv1.Wait_KIND_CONNECTING, tv1.Wait_KIND_DISK_FULL:
	}
	return nil
}

type volumeAt struct {
	node   role.NodeID
	volume string
}

// space is what t sends now and its NeedsSpace verdict, first come,
// first served on its volume. Its missing files reserve what they still
// have to copy in position order, its receiver's verified bytes counted
// off the first, up to the first file that doesn't fit: those files are
// sent, and the rest wait for room. t needs space only when its first
// missing file doesn't fit, or while its receiver found the disk full and
// no volume sample is newer than that report.
func (s snapshot) space(t Transfer, d dest, recv *tv1.Receiving, reserved map[volumeAt]int64) ([]File, *Space) {
	files := slices.DeleteFunc(slices.Clone(t.Files), func(f File) bool { return !missing(f) })
	if d.root == nil || d.root.Volume == nil {
		return files, nil
	}
	vol := d.root.Volume
	at := volumeAt{d.node, vol.ID}
	room := vol.Free - s.levels.For(d.node, vol.Mount, vol.Total).Error.Bytes - reserved[at]
	credit := recv.GetVerifiedBytes()
	var fit, need int64
	sent := len(files)
	for i, f := range files {
		left := max(f.Size-credit, 0)
		credit = max(credit-f.Size, 0)
		need += left
		if sent == len(files) && fit+left > room {
			if i == 0 {
				return nil, &Space{Node: d.node, Volume: vol.Name, Need: fit + left - room}
			}
			sent = i
		}
		if sent == len(files) {
			fit += left
		}
	}
	if recv.GetWait().GetKind() == tv1.Wait_KIND_DISK_FULL {
		full, ok := s.fullAt[t.ID]
		if !ok {
			full = s.now
		}
		if !vol.SampledAt.After(full) {
			return nil, &Space{Node: d.node, Volume: vol.Name, Need: need}
		}
	}
	reserved[at] += fit
	return files[:sent], nil
}

// hold judges a line while the library is Reconciling: nothing on it is
// active, and each Transfer keeps its staging and waits.
func (j *judgment) hold(l *line) {
	for _, c := range l.cands {
		v := &j.views[c.i]
		v.State, v.Wait = Waiting, &Wait{Kind: Reconciling}
	}
}

// run judges one pair's line, held while the library is Reconciling: its first Transfer is active and every
// other waits behind it. One whose receiver reported a step-aside wait
// goes to the back, so alone on its pair it stays active and its receiver
// retries. A Low one goes behind every Normal one until it has waited
// LowWait, unless its receiver already runs it. Only a purpose that
// yields is preempted: it goes behind every Normal one even while its
// receiver runs it.
func (j *judgment) run(s snapshot, l *line) {
	if s.paused {
		j.hold(l)
		return
	}
	rank := func(c candidate) int {
		switch {
		case c.aside != nil:
			return 2
		case s.yields(c, l), s.waitsLow(c):
			return 1
		}
		return 0
	}
	slices.SortStableFunc(l.cands, func(a, b candidate) int { return rank(a) - rank(b) })
	for _, c := range l.cands[1:] {
		if s.waitsLow(c) {
			j.wake(c.t.Created.Add(s.lowWait))
		}
	}
	for ahead, c := range l.cands[1:] {
		v := &j.views[c.i]
		if c.aside != nil {
			v.State, v.Wait = Waiting, c.aside
		} else {
			v.State, v.Ahead = Queued, ahead+1
		}
	}
	c := l.cands[0]
	v := &j.views[c.i]
	if w := s.pairWait(c.t.Source, c.d.node); w != nil {
		v.State, v.Wait = Waiting, w
		return
	}
	route, next := s.route(c, l.key)
	j.wake(next)
	v.Route = route
	switch {
	case route == tv1.Route_ROUTE_BOTH_DIAL:
		v.State, v.Wait = Waiting, &Wait{Kind: NoRoute, Node: c.d.node}
	case c.aside != nil:
		v.State, v.Wait = Waiting, c.aside
	case c.recv.GetActive() && c.recv.GetRoute() == route && c.recv.GetMoving():
		v.State, v.Rate = Copying, c.recv.GetRate()
		if v.Rate > 0 {
			v.Estimate = time.Duration((v.Total-v.Verified)/v.Rate) * time.Second
		}
	default:
		v.State, v.Wait = Waiting, &Wait{Kind: Connecting, Node: c.t.Source}
	}
	j.activate(s, c, route)
}

// yields is whether c's purpose yields and a Normal Transfer waits on its
// pair.
func (s snapshot) yields(c candidate, l *line) bool {
	return s.purposes[c.t.Purpose].Yields && slices.ContainsFunc(l.cands, func(o candidate) bool { return o.t.Priority == Normal })
}

// waitsLow is whether c is a Low Transfer that still waits behind Normal
// ones: younger than LowWait, and not running at its receiver.
func (s snapshot) waitsLow(c candidate) bool {
	return c.t.Priority == Low && s.now.Before(c.t.Created.Add(s.lowWait)) && !c.recv.GetActive()
}

// wake makes at the instant the answer changes next, when it is earlier.
func (j *judgment) wake(at time.Time) {
	if !at.IsZero() && (j.next.IsZero() || at.Before(j.next)) {
		j.next = at
	}
}

// pairWait is the wait an end of a pair gives every Transfer on it: Node
// offline or lapsed, or on another build. A Node reconnecting stops
// nothing.
func (s snapshot) pairWait(ends ...role.NodeID) *Wait {
	for _, n := range ends {
		e, ok := s.ends[n]
		switch {
		case !ok, e.Standing == nodes.Offline, e.Standing == nodes.LapsedNode:
			return &Wait{Kind: NodeAway, Node: n}
		case e.Standing == nodes.MismatchedNode:
			return &Wait{Kind: NodeOtherBuild, Node: n}
		}
	}
	return nil
}

// route is the active Transfer's route: the first on the ladder whose
// dialer hasn't given it up, starting from the pair's last route while
// that moved bytes within RouteRetry. It also returns when that start
// expires, zero for never. No route left between two Workers is
// BOTH_DIAL. A dialer's gave_up counts only from facts on that route or a
// later one, so facts from before a flip can't skip a route.
func (s snapshot) route(c candidate, key pairKey) (tv1.Route, time.Time) {
	ladder := []tv1.Route{tv1.Route_ROUTE_RECEIVER_DIALS, tv1.Route_ROUTE_SENDER_DIALS}
	if s.ends[key.source].Controller || s.ends[key.dest].Controller {
		ladder = append(ladder, tv1.Route_ROUTE_NODE_LINK)
	}
	var next time.Time
	start := 0
	if row, ok := s.pairs[key]; ok && row.Route != nil && *row.Route > 1 {
		if expiry := row.RouteAt.Time().Add(s.routeRetry); s.now.Before(expiry) {
			if i := slices.Index(ladder, tv1.Route(*row.Route)); i >= 0 { //nolint:gosec // G115: the column checks it is 1, 2 or 3
				start, next = i, expiry
			}
		}
	}
	send := s.sending(c.t)
	for _, r := range ladder[start:] {
		if !gaveUp(r, c.recv, send) {
			return r, next
		}
	}
	return tv1.Route_ROUTE_BOTH_DIAL, next
}

func gaveUp(r tv1.Route, recv *tv1.Receiving, send *tv1.Sending) bool {
	if r == tv1.Route_ROUTE_SENDER_DIALS {
		return send.GetRoute() >= r && slices.Contains(send.GetGaveUp(), r)
	}
	return recv.GetActive() && recv.GetRoute() >= r && slices.Contains(recv.GetGaveUp(), r)
}

// activate puts the active Transfer in both ends' intents.
func (j *judgment) activate(s snapshot, c candidate, route tv1.Route) {
	files := make([]*tv1.File, len(c.files))
	for i, f := range c.files {
		files[i] = &tv1.File{Position: uint32(f.Position), Source: sourceOf(f.Source), Size: f.Size} //nolint:gosec // G115: a position counts a Request's files
	}
	send := j.intent(s, c.t.Source)
	send.Active = append(send.Active, &tv1.Active{
		TransferId: c.t.ID.Bytes(), Side: tv1.Side_SIDE_SEND, Peer: s.peer(c.d.node), Route: route, Files: files,
	})
	recv := j.intent(s, c.d.node)
	recv.Active = append(recv.Active, &tv1.Active{
		TransferId: c.t.ID.Bytes(), Side: tv1.Side_SIDE_RECEIVE, Peer: s.peer(c.t.Source), Route: route, Files: files,
		Destination: c.d.wire(c.t),
	})
}

// peer is how the other end reaches node. A Worker reaches its Controller
// where it keeps its link, so the Controller lists no addresses.
func (s snapshot) peer(node role.NodeID) *tv1.Peer {
	p := s.peers[node]
	out := &tv1.Peer{Key: p.Key[:]}
	if !s.ends[node].Controller {
		out.Addresses = p.Addresses
	}
	return out
}

func sourceOf(ref source.Ref) *tv1.Source {
	switch r := ref.(type) {
	case source.Torrent:
		return &tv1.Source{Ref: &tv1.Source_Torrent{Torrent: &tv1.TorrentPayload{InfoHash: r.Hash[:], FileIndex: uint32(r.File)}}} //nolint:gosec // G115: a torrent's file index fits libtorrent's int
	case source.Path:
		return &tv1.Source{Ref: &tv1.Source_Path{Path: &tv1.PathSource{Abs: r.Abs}}}
	case source.Backup:
		return &tv1.Source{Ref: &tv1.Source_Backup{Backup: &tv1.BackupPlace{Folder: r.Folder, Name: r.Name}}}
	case source.Library:
		return &tv1.Source{Ref: &tv1.Source_Library{Library: &tv1.LibrarySource{Root: r.Root[:], Rel: r.Rel}}}
	case source.Scratch:
		return &tv1.Source{Ref: &tv1.Source_Scratch{Scratch: r.Name}}
	}
	panic("transfers: a source of no known kind")
}
