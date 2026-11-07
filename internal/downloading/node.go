package downloading

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading/internal/db"
	"github.com/jatassi/slipstream/internal/downloading/internal/external"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/torrent/clients/transmission"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// openers opens a session with each kind this Node reaches.
var openers = map[clients.Kind]func(clients.Session) clients.Client{
	clients.Transmission: transmission.Open,
}

var kindFromWire = map[dlv1.Kind]clients.Kind{
	dlv1.Kind_KIND_QBITTORRENT:  clients.QBittorrent,
	dlv1.Kind_KIND_TRANSMISSION: clients.Transmission,
	dlv1.Kind_KIND_DELUGE:       clients.Deluge,
	dlv1.Kind_KIND_RTORRENT:     clients.RTorrent,
}

type NodeConfig struct {
	// HTTP is the Node's client, which the Fake world dials through its
	// fronts.
	HTTP    *http.Client
	Timings Timings
	// Folder judges a client's download folder as this Node sees it:
	// there, readable and apart from every root this Node holds, and
	// writable too when write is set.
	Folder func(ctx context.Context, path string, write bool) (storing.CheckResult, error)
	// Rand is the stream every actor draws its backoff from.
	Rand *rand.Rand
	Log  *slog.Logger
}

// Node is a Node's Downloading Role: its built-in client and one actor per
// external client in its intent. It is the Role's holder, the
// DownloadingService and the file seams. It keeps no torrent state: it
// routes by the table it rebuilds from each intent.
type Node struct {
	builtin *Client
	cfg     NodeConfig
	routes  atomic.Pointer[routes]

	mu     sync.Mutex
	actors map[storage.UUID]*external.Actor
	local  *storage.DB[storage.Local]

	watching *nodeWatching
}

// routes is which external client holds each Torrent of one intent, by
// Torrent id and by info hash. A Torrent in neither is the built-in
// client's. The Node key makes byHash a function.
type routes struct {
	byID   map[storage.UUID]storage.UUID
	byHash map[bencode.InfoHash]storage.UUID
}

func NewNode(builtin *Client, cfg NodeConfig) *Node {
	cfg.Rand = rand.New(&lockedSource{r: cfg.Rand}) //nolint:gosec // the Node's seeded stream, shared by the actors
	n := &Node{builtin: builtin, cfg: cfg, actors: map[storage.UUID]*external.Actor{}, watching: newNodeWatching()}
	n.routes.Store(&routes{})
	return n
}

// Run is the Role's run function. It gives the built-in client a Holder
// whose intent lists only its own torrents and discards, starts an actor
// for each client the intent adds, hands each its share, stops each one
// the intent drops, forgetting its local rows, and swaps the routing
// table. Every actor reports through h with local.
func (n *Node) Run(local *storage.DB[storage.Local], ctl Control) func(context.Context, Holder) error {
	return func(ctx context.Context, h Holder) error {
		n.mu.Lock()
		n.local = local
		n.mu.Unlock()
		in := make(chan role.Snapshot[*dlv1.Intent], 1)
		var wg sync.WaitGroup
		var err error
		wg.Go(func() { err = n.builtin.Run(local, ctl)(ctx, filtered{Holder: h, in: in}) })
		for s := range h.Intent(ctx) {
			mine, shares := split(s)
			n.routes.Store(routesOf(s.Intent))
			select {
			case <-in:
			default:
			}
			in <- mine
			n.sync(ctx, shares, local, ctl, h)
		}
		close(in)
		wg.Wait()
		n.mu.Lock()
		actors := n.actors
		n.actors = map[storage.UUID]*external.Actor{}
		n.mu.Unlock()
		for _, a := range actors {
			a.Halt()
		}
		return err
	}
}

// split is s's built-in snapshot, its torrents and discards with no
// client, and each external client's share.
func split(s role.Snapshot[*dlv1.Intent]) (mine role.Snapshot[*dlv1.Intent], shares map[storage.UUID]external.Share) {
	in := s.Intent
	builtin := &dlv1.Intent{
		Settings: in.GetSettings(), HoldDownloads: in.GetHoldDownloads(), Thresholds: in.GetThresholds(),
		SampleEverySeconds: in.GetSampleEverySeconds(),
	}
	byClient := map[storage.UUID]*external.Share{}
	for _, c := range in.GetClients() {
		if id, err := storage.UUIDFromBytes(c.GetId()); err == nil {
			byClient[id] = &external.Share{
				Client:          c,
				StalledAfter:    time.Duration(in.GetSettings().GetStalledAfterSeconds()) * time.Second,
				UnregisteredGap: time.Duration(in.GetThresholds().GetUnregisteredGapSeconds()) * time.Second,
				MayStart:        s.MayStart(),
			}
		}
	}
	for _, t := range in.GetTorrents() {
		if len(t.GetClientId()) == 0 {
			builtin.Torrents = append(builtin.Torrents, t)
		} else if sh := shareOf(byClient, t.GetClientId()); sh != nil {
			sh.Torrents = append(sh.Torrents, t)
		}
	}
	for _, d := range in.GetDiscards() {
		if len(d.GetClientId()) == 0 {
			builtin.Discards = append(builtin.Discards, d)
		} else if sh := shareOf(byClient, d.GetClientId()); sh != nil {
			sh.Discards = append(sh.Discards, d)
		}
	}
	shares = make(map[storage.UUID]external.Share, len(byClient))
	for id, sh := range byClient {
		shares[id] = *sh
	}
	return role.Snapshot[*dlv1.Intent]{Intent: builtin, Standing: s.Standing, Draining: s.Draining}, shares
}

func shareOf(shares map[storage.UUID]*external.Share, id []byte) *external.Share {
	u, err := storage.UUIDFromBytes(id)
	if err != nil {
		return nil
	}
	return shares[u]
}

func routesOf(in *dlv1.Intent) *routes {
	r := &routes{byID: map[storage.UUID]storage.UUID{}, byHash: map[bencode.InfoHash]storage.UUID{}}
	add := func(id, hash, client []byte) {
		c, err := storage.UUIDFromBytes(client)
		if err != nil {
			return
		}
		if u, err := storage.UUIDFromBytes(id); err == nil {
			r.byID[u] = c
		}
		r.byHash[bencode.InfoHash(hash)] = c
	}
	for _, t := range in.GetTorrents() {
		if len(t.GetClientId()) > 0 {
			add(t.GetId(), t.GetInfoHash(), t.GetClientId())
		}
	}
	for _, d := range in.GetDiscards() {
		if len(d.GetClientId()) > 0 {
			add(d.GetId(), d.GetInfoHash(), d.GetClientId())
		}
	}
	return r
}

// sync starts an actor for each new client of a kind this Node reaches,
// hands every other its share, and stops each one the intent dropped.
func (n *Node) sync(ctx context.Context, shares map[storage.UUID]external.Share, local *storage.DB[storage.Local], ctl Control, h Holder) {
	n.mu.Lock()
	var stopped []*external.Actor
	for id, a := range n.actors {
		if _, ok := shares[id]; !ok {
			stopped = append(stopped, a)
			delete(n.actors, id)
		}
	}
	updated := map[*external.Actor]external.Share{}
	for id, s := range shares {
		if a, ok := n.actors[id]; ok {
			updated[a] = s
			continue
		}
		kind := kindFromWire[s.Client.GetKind()]
		open, ok := openers[kind]
		if !ok {
			n.cfg.Log.Warn("this Node can't reach a Download Client of this kind yet", "client", s.Client.GetName(), "kind", string(kind))
			continue
		}
		n.actors[id] = external.Start(ctx, n.actorConfig(id, kind, open, local, ctl, h), s)
	}
	n.mu.Unlock()
	for a, s := range updated {
		a.Update(s)
	}
	for _, a := range stopped {
		if err := a.Stop(ctx); err != nil && ctx.Err() == nil {
			n.cfg.Log.Error("forget the torrents of a Download Client the Node no longer runs", "err", err.Error())
		}
	}
}

// actorConfig is one client's actor's configuration.
func (n *Node) actorConfig(id storage.UUID, kind clients.Kind, open func(clients.Session) clients.Client, local *storage.DB[storage.Local], ctl Control, h Holder) external.Config {
	t := n.cfg.Timings
	return external.Config{
		ID: id, Kind: kind, Open: open, HTTP: n.cfg.HTTP, Subfolder: folderName, Links: links,
		Timings: external.Timings{
			RetryMin: t.ClientRetryMin, RetryMax: t.ClientRetryMax, RefusedRetry: t.ClientRefusedRetry, Grace: t.ClientGrace,
			ResendWait: t.ClientResendWait, MissingShare: t.ClientMissingShare, MissingFloor: t.ClientMissingFloor,
			Trackers: t.ClientTrackers, Call: t.ClientCall, LiveFloor: t.ClientLiveFloor, LiveSample: t.LiveSample, TotalsReport: t.TotalsReport,
		},
		Local: local, Holder: h, Control: ctl,
		Rand: n.cfg.Rand,
		Log:  n.cfg.Log.With("client", id.String(), "kind", string(kind)),
	}
}

func (n *Node) actor(id storage.UUID) *external.Actor {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.actors[id]
}

func (n *Node) actorList() []*external.Actor {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make([]*external.Actor, 0, len(n.actors))
	for _, a := range n.actors {
		out = append(out, a)
	}
	return out
}

// filtered is the built-in client's Holder: the intent from in, its
// latest snapshot only, and facts and reports straight to the Role's.
type filtered struct {
	Holder
	in <-chan role.Snapshot[*dlv1.Intent]
}

func (f filtered) Intent(ctx context.Context) iter.Seq[role.Snapshot[*dlv1.Intent]] {
	return func(yield func(role.Snapshot[*dlv1.Intent]) bool) {
		for {
			select {
			case s, ok := <-f.in:
				if !ok || !yield(s) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}
}

// Release releases the built-in client and forgets what the Node kept of
// its external clients' torrents, keeping every torrent in its client.
func (n *Node) Release(ctx context.Context) error {
	err := n.builtin.Release(ctx)
	n.routes.Store(&routes{})
	n.mu.Lock()
	local := n.local
	n.mu.Unlock()
	if local == nil {
		return err
	}
	return errors.Join(err, local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).DeleteExternals(ctx)
	}))
}

// held is which actor the intent says holds Torrent id, false for the
// built-in client.
func (n *Node) held(id []byte) (storage.UUID, *external.Actor, bool) {
	u, err := storage.UUIDFromBytes(id)
	if err != nil {
		return storage.UUID{}, nil, false
	}
	c, ok := n.routes.Load().byID[u]
	if !ok {
		return u, nil, false
	}
	return u, n.actor(c), true
}

// Inspect is the built-in client's, or the external client's that holds
// the torrent.
func (n *Node) Inspect(ctx context.Context, req *dlv1.InspectRequest) (*dlv1.InspectResponse, error) {
	id, a, ok := n.held(req.GetId())
	if !ok {
		return n.builtin.Inspect(ctx, req)
	}
	var res *dlv1.InspectResponse
	if a != nil {
		res = a.Inspect(ctx, id)
	}
	if res == nil {
		return nil, rpc.Errorf(rpc.CodeNotFound, "this Node holds no torrent %s", id)
	}
	return res, nil
}

var clientActs = map[dlv1.Act]clients.Act{dlv1.Act_ACT_ANNOUNCE: clients.Announce, dlv1.Act_ACT_RECHECK: clients.Recheck}

// Act splits req's ids by the client that holds each, and answers in the
// order asked.
func (n *Node) Act(ctx context.Context, req *dlv1.ActRequest) (*dlv1.ActResponse, error) {
	act, ok := clientActs[req.GetAct()]
	if !ok {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "no act %v", req.GetAct())
	}
	results := make([]*dlv1.ActResult, len(req.GetIds()))
	err := n.byClient(req.GetIds(), func(ids [][]byte, at []int) error {
		res, err := n.builtin.Act(ctx, &dlv1.ActRequest{Act: req.GetAct(), Ids: ids})
		for i, r := range res.GetResults() {
			results[at[i]] = r
		}
		return err
	}, func(a *external.Actor, ids []storage.UUID, at []int) {
		for i, r := range a.Act(ctx, act, ids) {
			results[at[i]] = r
		}
	})
	if err != nil {
		return nil, err
	}
	return &dlv1.ActResponse{Results: results}, nil
}

// Frees splits req's ids by the client that holds each, and answers in
// the order asked.
func (n *Node) Frees(ctx context.Context, req *dlv1.FreesRequest) (*dlv1.FreesResponse, error) {
	freed := make([]*dlv1.Freed, len(req.GetIds()))
	err := n.byClient(req.GetIds(), func(ids [][]byte, at []int) error {
		res, err := n.builtin.Frees(ctx, &dlv1.FreesRequest{Ids: ids})
		for i, f := range res.GetFreed() {
			freed[at[i]] = f
		}
		return err
	}, func(a *external.Actor, ids []storage.UUID, at []int) {
		for i, f := range a.Frees(ctx, ids) {
			freed[at[i]] = f
		}
	})
	if err != nil {
		return nil, err
	}
	return &dlv1.FreesResponse{Freed: freed}, nil
}

// byClient runs builtin on the ids the built-in client holds and theirs
// on each actor's, with each id's place in ids. An id whose client has no
// actor is answered by builtin, which holds no such torrent.
func (n *Node) byClient(ids [][]byte, builtin func(ids [][]byte, at []int) error, theirs func(a *external.Actor, ids []storage.UUID, at []int)) error {
	type group struct {
		ids []storage.UUID
		at  []int
	}
	var mine [][]byte
	var mineAt []int
	groups := map[*external.Actor]*group{}
	var order []*external.Actor
	for i, raw := range ids {
		u, err := storage.UUIDFromBytes(raw)
		c, ok := n.routes.Load().byID[u]
		a := n.actor(c)
		if err != nil || !ok || a == nil {
			mine, mineAt = append(mine, raw), append(mineAt, i)
			continue
		}
		g := groups[a]
		if g == nil {
			g = &group{}
			groups[a] = g
			order = append(order, a)
		}
		g.ids, g.at = append(g.ids, u), append(g.at, i)
	}
	for _, a := range order {
		theirs(a, groups[a].ids, groups[a].at)
	}
	if len(mine) == 0 {
		return nil
	}
	return builtin(mine, mineAt)
}

func (n *Node) Sample(ctx context.Context, req *dlv1.SampleRequest) (*dlv1.SampleResponse, error) {
	return n.builtin.Sample(ctx, req)
}

func (n *Node) Resolve(ctx context.Context, req *dlv1.ResolveRequest) (*dlv1.ResolveResponse, error) {
	return n.builtin.Resolve(ctx, req)
}

// Path is the external client's that holds h, the built-in client's for
// any other.
func (n *Node) Path(h bencode.InfoHash, index int) (string, error) {
	c, ok := n.routes.Load().byHash[h]
	if !ok {
		return n.builtin.Path(h, index)
	}
	a := n.actor(c)
	if a == nil {
		return "", ErrNotPresent
	}
	p, ok := a.Path(h, index)
	if !ok {
		return "", ErrNotPresent
	}
	return p, nil
}

// Open is the external client's that holds the torrent, the built-in
// client's for any other.
func (n *Node) Open(ref source.Ref) (string, error) {
	r, ok := ref.(source.Torrent)
	if !ok {
		return n.builtin.Open(ref)
	}
	c, ok := n.routes.Load().byHash[r.Hash]
	if !ok {
		return n.builtin.Open(ref)
	}
	a := n.actor(c)
	if a == nil {
		return "", &source.Absent{Reason: source.NotReady, Detail: "this Node can't reach the torrent's Download Client yet"}
	}
	return a.Open(r)
}

// Folders is the built-in client's folders and each external client's
// download folder as this Node sees it, which no root may overlap.
func (n *Node) Folders() []string {
	out := slices.Clone(n.builtin.Folders())
	for _, a := range n.actorList() {
		if f, ok := a.Folder(); ok && !slices.Contains(out, f) {
			out = append(out, f)
		}
	}
	slices.Sort(out[1:])
	return out
}

// TestClient opens a session of its own for req's draft and tests it,
// then judges its download folder, the client's default when none is set,
// as this Node sees it through the draft's Client folders, and adds the
// warnings only the Import delay decides. It never touches an actor.
func (n *Node) TestClient(ctx context.Context, req *dlv1.TestClientRequest) (*dlv1.TestClientResponse, error) {
	c := req.GetClient()
	kind, ok := kindFromWire[c.GetKind()]
	if !ok {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "no Download Client kind %v", c.GetKind())
	}
	open, ok := openers[kind]
	if !ok {
		return nil, rpc.Errorf(rpc.CodeUnimplemented, "this Node can't reach a %s client yet", kind.Facts().Name)
	}
	session, label, folders, err := n.draft(kind, c, req.GetSecrets())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	client := open(session)
	defer func() { _ = client.Close(context.WithoutCancel(ctx)) }()
	tested, err := client.Test(ctx, label)
	if err != nil {
		return failedTest(err)
	}
	folder := c.GetFolder()
	if folder == "" {
		if folder, err = client.DefaultFolder(ctx, label); err != nil {
			return failedTest(err)
		}
	}
	onNode, problem, err := n.judgeFolder(ctx, kind, folders, folder)
	switch {
	case err != nil:
		return nil, err
	case problem != nil:
		return &dlv1.TestClientResponse{Answer: &dlv1.TestClientResponse_Failed{Failed: external.ProblemOnWire(problem)}}, nil
	}
	passed := &dlv1.Passed{Product: kind.Facts().Name, Version: tested.Version.String(), NodeFolder: onNode}
	for _, w := range warningsOf(kind, tested, c.GetImportDelaySeconds()) {
		passed.Warnings = append(passed.Warnings, warningOnWire[w])
	}
	for _, d := range tested.Daemons {
		passed.Daemons = append(passed.Daemons, &dlv1.Daemon{Host: d.Host, Port: int32(d.Port)}) //nolint:gosec // a port fits
	}
	return &dlv1.TestClientResponse{Answer: &dlv1.TestClientResponse_Passed{Passed: passed}}, nil
}

// draft is the session, label and Client folders req's draft names.
func (n *Node) draft(kind clients.Kind, c *dlv1.Client, secrets *dlv1.Secrets) (clients.Session, clients.Label, clients.Folders, error) {
	address, err := clients.ParseAddress(c.GetAddress(), kind)
	if err != nil {
		return clients.Session{}, clients.Label{}, clients.Folders{}, fmt.Errorf("the address: %w", err)
	}
	label, err := clients.ParseLabel(c.GetLabel(), kind)
	if err != nil {
		return clients.Session{}, clients.Label{}, clients.Folders{}, fmt.Errorf("the label: %w", err)
	}
	pairs := make([]clients.Folder, len(c.GetClientFolders()))
	for i, f := range c.GetClientFolders() {
		pairs[i] = clients.Folder{Client: f.GetClient(), Node: f.GetNode()}
	}
	folders, err := clients.ParseFolders(pairs)
	if err != nil {
		return clients.Session{}, clients.Label{}, clients.Folders{}, err
	}
	s := clients.Session{
		HTTP: n.cfg.HTTP, Address: address, Username: c.GetUsername(),
		Secrets:      clients.Secrets{Password: secrets.GetPassword(), APIKey: secrets.GetApiKey()},
		TrackerEvery: n.cfg.Timings.ClientTrackers, Call: n.cfg.Timings.ClientCall, Log: n.cfg.Log,
	}
	if d := c.GetDaemon(); d != nil {
		s.Daemon = &clients.Daemon{Host: d.GetHost(), Port: int(d.GetPort())}
	}
	return s, label, folders, nil
}

// failedTest is a Test that failed with err: a Problem is the Test's
// answer, any other error the call's.
func failedTest(err error) (*dlv1.TestClientResponse, error) {
	p, ok := errors.AsType[*clients.Problem](err)
	if !ok {
		return nil, rpc.Errorf(rpc.CodeUnavailable, "test the client: %v", err)
	}
	return &dlv1.TestClientResponse{Answer: &dlv1.TestClientResponse_Failed{Failed: external.ProblemOnWire(p)}}, nil
}

// judgeFolder is folder, as the client reports it, on this Node, or the
// problem that keeps the Node from reading it there: no Client folder
// holds it, it isn't there or can't be read, or it overlaps a library
// root. A kind whose remove never deletes data needs it writable too.
func (n *Node) judgeFolder(ctx context.Context, kind clients.Kind, folders clients.Folders, folder string) (string, *clients.Problem, error) {
	onNode, used, ok := folders.OnNode(folder, filepath.Separator)
	if !ok {
		return "", &clients.Problem{Cause: clients.FolderUnreadable, Folder: folder}, nil
	}
	res, err := n.cfg.Folder(ctx, onNode, !kind.Facts().DeletesData)
	if err != nil {
		return "", nil, rpc.Errorf(rpc.CodeUnavailable, "check the download folder on this Node: %v", err)
	}
	switch r := res.Refusal; {
	case r == nil:
		return res.Path, nil, nil
	case r.Reason == storing.ReasonInsideRoot || r.Reason == storing.ReasonContainsRoot:
		return "", &clients.Problem{Cause: clients.FolderInRoot, Folder: onNode}, nil
	}
	return "", &clients.Problem{Cause: clients.FolderUnreadable, Folder: onNode, Paired: used.Client}, nil
}

// warningsOf is a passing Test's warnings, with the two the Import delay
// decides: a client that moves finished files, and a kind that can't
// report a move, each only while the delay is 0.
func warningsOf(kind clients.Kind, tested clients.Tested, delaySeconds int64) []clients.Warning {
	out := slices.Clone(tested.Warnings)
	if delaySeconds == 0 && tested.MovesAfterFinish {
		out = append(out, clients.MovesAfterFinish)
	}
	if delaySeconds == 0 && kind.Facts().NoMovingState {
		out = append(out, clients.NoMovingState)
	}
	return out
}

var warningOnWire = map[clients.Warning]dlv1.Warning{
	clients.RemovesAtLimit:   dlv1.Warning_WARNING_REMOVES_AT_LIMIT,
	clients.MovesAfterFinish: dlv1.Warning_WARNING_MOVES_AFTER_FINISH,
	clients.NoMovingState:    dlv1.Warning_WARNING_NO_MOVING_STATE,
}

// CheckClients has each named actor poll now and answers once each one
// has committed its ClientPoll with req's token.
func (n *Node) CheckClients(ctx context.Context, req *dlv1.CheckClientsRequest) (*dlv1.CheckClientsResponse, error) {
	var actors []*external.Actor
	for _, raw := range req.GetIds() {
		id, err := storage.UUIDFromBytes(raw)
		if err != nil {
			return nil, rpc.Errorf(rpc.CodeInvalidArgument, "client id: %v", err)
		}
		a := n.actor(id)
		if a == nil {
			return nil, rpc.Errorf(rpc.CodeFailedPrecondition, "this Node runs no Download Client %s", id)
		}
		actors = append(actors, a)
	}
	errs := make([]error, len(actors))
	var wg sync.WaitGroup
	for i, a := range actors {
		wg.Go(func() { errs[i] = a.Check(ctx, req.GetCheck()) })
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return nil, rpc.Errorf(rpc.CodeUnavailable, "check the Download Clients: %v", err)
	}
	return &dlv1.CheckClientsResponse{}, nil
}

// lockedSource lets every actor's goroutine draw from one stream.
type lockedSource struct {
	mu sync.Mutex
	r  *rand.Rand
}

func (l *lockedSource) Uint64() uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.r.Uint64()
}
