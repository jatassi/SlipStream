package grabs

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// outcome is what a step decided. A step does its IO outside any Write;
// commit applies its outcome in one Write.
type outcome interface{ outcome() }

// advance moves a Grab to its next stage with what that stage needs: the
// .torrent and the Nodes that fetched and download it into downloading.
type advance struct {
	to      Stage
	torrent *dotTorrent
	fetched *fetched
}

// dotTorrent is a fetched .torrent, decoded and as the Indexer sent it.
type dotTorrent struct {
	metainfo bencode.Metainfo
	raw      []byte
}

// fetched is the Searching Node that fetched a Grab's .torrent and the
// Downloading Node that takes it.
type fetched struct{ by, to role.NodeID }

type fail struct {
	code Code
	msg  string
}

// wait retries the same step once a backoff has passed and, when the
// Grab's Node is out of step or roster is set, the roster has changed; when
// root is set, once it is closed or the roster has changed. When rerouted
// is set and closes first, it retries at once. When until is set, it
// retries at once at until or the Indexers' next change. It never fails
// the Grab.
type wait struct {
	cause    error
	roster   bool
	root     <-chan struct{}
	rerouted <-chan struct{}
	until    time.Time
}

// stopped commits nothing: the step runs again at the next start.
type stopped struct{}

func (advance) outcome() {}
func (fail) outcome()    {}
func (wait) outcome()    {}
func (stopped) outcome() {}

// steps is the pipeline: the step each stage runs. An importing Grab has
// none: its Import moves it through Follow. Imported and Failed have none.
var steps = map[Stage]func(*Grabs, context.Context, Grab) outcome{
	Fetching:    (*Grabs).fetch,
	Downloading: (*Grabs).downloaded,
}

// due reports whether g has a step to run now. A downloading Grab waits
// while its Torrent is Pending, which the progress tick reads again.
func (g Grab) due() bool { return steps[g.Stage] != nil && !g.waitsOnTorrent() }

func (g Grab) waitsOnTorrent() bool {
	return g.Stage == Downloading && g.Torrent.Outcome == downloadclients.Outcome(downloadclients.Pending{})
}

// classify turns a remote or file step's error into its outcome: a link
// that is down, mismatched or lost the call waits, and so does a torrent
// the client has not re-added yet; NotHolder means the Node lost the Role;
// any other error fails the Grab with code.
func classify(ctx context.Context, err error, code Code) outcome {
	why, notSent := rpc.NotSent(err)
	switch {
	case ctx.Err() != nil:
		return stopped{}
	case notSent && why == rpc.NotHolder:
		return fail{IndexerUnavailable, detail(err)}
	case notSent, rpc.Lost(err), errors.Is(err, downloading.ErrNotPresent):
		return wait{cause: err}
	}
	return fail{code, detail(err)}
}

// detail is a remote step's error as its failure's Detail: the cause the
// remote Node gave, without the call's code or the wrapping the failure's
// code already says.
func detail(err error) string {
	if u, ok := errors.AsType[*indexers.UnavailableError](err); ok {
		return u.Reason()
	}
	return rpc.MessageOf(err)
}

// local is the outcome of an error reading or writing the library inside
// a step: never a failure of the Grab.
func local(ctx context.Context, err error) outcome {
	if ctx.Err() != nil {
		return stopped{}
	}
	return wait{cause: err}
}

// fetch brings the .torrent through the Node the Indexer routes to now, or
// a magnet link's metadata through the Node of the client the Grab routes
// to, and hands it to that client: by default the one on the Node holding
// the Work's root when it may download, so the file lands by a hard link.
// While no Node may take either, or the client's Node is out of step, it
// waits for the roster or the Grab's route to change.
func (gr *Grabs) fetch(ctx context.Context, g Grab) outcome {
	if w := gr.reconcilingWait(ctx); w != nil {
		return w
	}
	if g.Ticket.Indexer.IsZero() {
		return dropOf[indexers.DropRemoved]
	}
	if _, w := gr.route(ctx, g); w != nil {
		return w
	}
	link, err := gr.link(g)
	if err != nil {
		return local(ctx, err)
	}
	got, err := gr.cfg.Indexers.Fetch(ctx, g.Ticket, link)
	if errors.Is(err, bencode.ErrV2Only) {
		return fail{TorrentV2Only, v2OnlyWords}
	}
	if _, invalid := errors.AsType[*magnet.InvalidError](err); invalid {
		return fail{MagnetInvalid, err.Error()}
	}
	if drop, ok := errors.AsType[*indexers.DropError](err); ok {
		return dropOf[drop.Why]
	}
	if w, ok := errors.AsType[*indexers.WaitError](err); ok {
		return wait{cause: err, roster: w.Outcome.Kind == indexers.Waiting, until: w.Outcome.Until}
	}
	if err != nil {
		return classify(ctx, err, TorrentUnreachable)
	}
	to, w := gr.route(ctx, g)
	if w != nil {
		return w
	}
	switch got := got.(type) {
	case indexers.Torrent:
		return decoded(g, got.Raw, fetched{by: got.Node, to: to.Node})
	case indexers.Magnet:
		return gr.resolve(ctx, g, got.Link, to.Node)
	}
	panic("grabs: unknown fetch result")
}

const v2OnlyWords = "The .torrent file has no BitTorrent v1 info hash."

// decoded is the advance to Downloading with the .torrent raw, or the
// failure that keeps raw from being g's.
func decoded(g Grab, raw []byte, f fetched) outcome {
	m, err := bencode.DecodeMetainfo(raw)
	switch {
	case errors.Is(err, bencode.ErrV2Only):
		return fail{TorrentV2Only, v2OnlyWords}
	case err != nil:
		return fail{TorrentInvalid, "The Indexer sent a .torrent file SlipStream can't read."}
	case g.Release.InfoHash != nil && *g.Release.InfoHash != m.InfoHash:
		return fail{InfohashMismatch, fmt.Sprintf("The .torrent file's info hash is %s, but the Indexer listed %s.", m.InfoHash, g.Release.InfoHash)}
	}
	return advance{to: Downloading, torrent: &dotTorrent{metainfo: m, raw: raw}, fetched: &f}
}

func (gr *Grabs) downloaded(_ context.Context, g Grab) outcome {
	switch o := g.Torrent.Outcome.(type) {
	case downloadclients.Completed:
		return advance{to: Importing}
	case downloadclients.Failed:
		return fail{DownloadFailed, o.Error}
	case downloadclients.Gone:
		return failOfGone(o)
	case downloadclients.Errored:
		return fail{ClientError, fmt.Sprintf("%s on %s reported an error for more than %s.",
			g.Torrent.Client.Name, g.Torrent.ClientError.Node, setting.Words(o.After))}
	}
	panic("grabs: a Grab whose Torrent is Pending is never due")
}

const writtenOffWords = "An Admin had SlipStream stop waiting for the Node downloading this Grab."

func failOfGone(g downloadclients.Gone) fail {
	switch g.Reason {
	case downloadclients.ByAdmin:
		return fail{Removed, "An Admin removed this Grab's torrent."}
	case downloadclients.ByMarkFailed:
		if g.By == "" {
			return fail{MarkedFailed, "An app using the API key marked its download failed."}
		}
		return fail{MarkedFailed, g.By + " marked its download failed."}
	case downloadclients.NodeWrittenOff:
		return fail{NodeWrittenOff, writtenOffWords}
	case downloadclients.ByClient:
		return fail{ClientRemoved, g.Client + " removed the torrent before SlipStream imported it."}
	case downloadclients.ClientDeleted:
		return fail{ClientDeleted, "An Admin deleted " + g.Client + " before SlipStream imported this torrent."}
	case downloadclients.GoalMet:
		return fail{GoalMet, "SlipStream removed this Grab's torrent once it met its seeding goal."}
	case downloadclients.ReleaseFaulted:
		return failOfFault(g)
	case downloadclients.Canceled:
		return fail{Canceled, "This Grab's torrent was removed when another Grab of it was canceled."}
	case downloadclients.Forgotten:
		return fail{Restored, "No Node had this Grab's torrent after the restore, and an Admin chose to forget it."}
	case downloadclients.Moved:
		panic("grabs: a Move's switch repoints its source's Grabs in the Write that removes it")
	}
	panic("grabs: no failure for removal reason " + string(g.Reason))
}

// failOfFault is how a downloading Grab fails when its Torrent is removed
// for a release fault. An engine error keeps failing it download_failed,
// as it did before the Blocklist.
func failOfFault(g downloadclients.Gone) fail {
	switch *g.Fault {
	case downloadclients.FaultEngineError:
		return fail{DownloadFailed, cmp.Or(g.Client, "The built-in client") + " couldn't read this Grab's torrent."}
	case downloadclients.FaultUnregistered:
		return fail{ReleaseFault, "The tracker reports this Grab's torrent unregistered."}
	case downloadclients.FaultNoPayload:
		return fail{ReleaseFault, "This Grab's torrent never downloaded any data."}
	}
	panic("grabs: no failure for release fault " + string(*g.Fault))
}

// errLeft rolls back a commit whose Grab has left the stage it moves
// from, along with any Torrent its attach added.
var errLeft = errors.New("the Grab left the stage")

// commit applies a step's outcome o to g in one Write that compares on
// g's stage, so a repeated step converges, then publishes. It is the only
// code that runs a step's outcome, and move the only code that writes a
// stage. A fetch's commit attaches the Grab to its Torrent, and fails with
// downloadclients.ErrRemoving while that must wait. A Grab whose Work was
// deleted went with it, so its commit commits nothing.
func (gr *Grabs) commit(ctx context.Context, g Grab, o outcome) error {
	var params db.AdvanceParams
	var taken []Grab
	var reran *Search
	var re *redo
	if _, drop := o.(dropped); drop && g.Stage == Fetching {
		var err error
		if re, err = gr.rerunOf(ctx, g); err != nil {
			return err
		}
	}
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		o, err := confirm(ctx, tx, g, o)
		if err != nil {
			return err
		}
		params, err = gr.move(ctx, tx, g, o)
		if err != nil || g.Stage != Fetching {
			return err
		}
		taken, reran, err = gr.left(ctx, tx, g, o, re)
		return err
	})
	switch {
	case errors.Is(err, errBlocked):
		return gr.commit(ctx, g, dropped{Blocklisted, blockedWords})
	case errors.Is(err, errLeft), errors.Is(err, works.ErrNoWork):
		return nil
	case err != nil:
		return err
	}
	if !params.NodeID.IsZero() {
		g.Node = params.NodeID
	}
	gr.entered(ctx, g, params.Next)
	if reran != nil {
		gr.cfg.SearchChanged.Publish(ctx, SearchChanged{SearchID: reran.ID.String(), WorkID: reran.WorkID.String()})
	}
	for _, t := range taken {
		gr.cfg.Indexers.Taken(ctx, t.Ticket)
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: t.ID.String(), WorkID: t.WorkID.String()})
	}
	return nil
}

// confirm refuses a fetch whose .torrent the Blocklist holds with
// errBlocked, before anything reaches a client. It re-reads a Grab's
// Torrent in tx, so a removal that committed after the step read it fails
// a downloading Grab by its reason rather than advancing it.
func confirm(ctx context.Context, tx *storage.Tx[file], g Grab, o outcome) (outcome, error) {
	if a, ok := o.(advance); ok && a.torrent != nil {
		hit, err := blockedHash(ctx, tx, a.torrent.metainfo.InfoHash)
		if err != nil {
			return nil, err
		}
		if hit {
			return nil, errBlocked
		}
	}
	if g.Stage != Downloading {
		return o, nil
	}
	now, err := downloadclients.Outcomes(ctx, tx, []downloadclients.ID{g.Torrent.ID})
	if err != nil {
		return nil, err
	}
	if gone, ok := now[g.Torrent.ID].Outcome.(downloadclients.Gone); ok {
		return failOfGone(gone), nil
	}
	return o, nil
}

// attach binds g to the Torrent of a's .torrent, in the client g routes
// to now unless another Node already holds it, and records the torrent's
// privacy and a private torrent's announce hosts on g's Tracker.
func (gr *Grabs) attach(ctx context.Context, tx *storage.Tx[file], g Grab, a advance, now storage.Time) (downloadclients.ID, role.NodeID, error) {
	tree, err := gr.cfg.Works.TreeIn(ctx, tx, g.WorkID)
	if err != nil {
		return downloadclients.ID{}, role.NodeID{}, err
	}
	target, err := routeOf(ctx, tx, gr.cfg.Router, g)
	if err != nil {
		return downloadclients.ID{}, role.NodeID{}, err
	}
	m := a.torrent.metainfo
	id, node, err := downloadclients.Attach(ctx, tx, downloadclients.Want{
		Target: target, Metainfo: m, Raw: a.torrent.raw, ContentType: tree.ContentType, Tracker: g.Ticket.Tracker,
		WrittenOff: func(node role.NodeID) bool { return gr.cfg.Roster.WrittenOff(role.Downloading, node) },
	}, now)
	if err != nil || g.Ticket.Tracker.IsZero() {
		return id, node, err
	}
	return id, node, indexers.AddHosts(ctx, tx, g.Ticket.Tracker, m.Info.Private, downloadclients.HostsOf(m.Trackers()))
}

// RepointTracker is grabs' indexers.Repoint: every Grab of from moves to
// into.
func RepointTracker(ctx context.Context, tx *storage.Tx[storage.Library], from, into indexers.TrackerID) error {
	return db.New(tx).RepointTracker(ctx, db.RepointTrackerParams{IntoID: into, FromID: from})
}

// RepointTorrentIn moves every Grab of Torrent from, at every stage, to
// Torrent to on node: a Move's switch.
func RepointTorrentIn(ctx context.Context, tx *storage.Tx[storage.Library], from, to downloadclients.ID, node role.NodeID) error {
	return db.New(tx).RepointAll(ctx, db.RepointAllParams{ToID: to, NodeID: node, UpdatedAt: storage.Now(), FromID: from})
}

// move writes o to g in tx, or fails with errLeft when g has left the
// stage it stood at when read. It is the only code that writes a Grab's
// stage: commit calls it for a step's outcome, cancelIn for a cancel and
// Follow for what the Grab's Import reports. A move into importing starts
// the Grab's Import in the same Write, a move into imported clears the
// failures of the cells it placed, and a failure records its cells'
// failures, except a drop at fetching, whose rerun decides first.
func (gr *Grabs) move(ctx context.Context, tx *storage.Tx[file], g Grab, o outcome) (db.AdvanceParams, error) {
	now := storage.Now()
	params, a := advanceParams(g, o, now)
	if a.torrent != nil {
		var err error
		if params.TorrentID, params.NodeID, err = gr.attach(ctx, tx, g, a, now); err != nil {
			return params, err
		}
	}
	q := db.New(tx)
	n, err := q.Advance(ctx, params)
	if err != nil {
		return params, err
	}
	if n == 0 {
		return params, errLeft
	}
	if err := q.InsertStage(ctx, db.InsertStageParams{GrabID: g.ID, Stage: params.Next, EnteredAt: now}); err != nil {
		return params, err
	}
	switch params.Next {
	case Importing:
		h, err := Handed(ctx, tx, g.ID)
		if err != nil {
			return params, err
		}
		return params, gr.importer.Start(ctx, tx, h)
	case Imported:
		return params, clearPlaced(ctx, tx, g)
	case Failed:
		if _, drop := o.(dropped); drop && g.Stage == Fetching {
			return params, nil
		}
		return params, gr.failed(ctx, tx, g, *params.FailureCode)
	case Fetching, Downloading:
	}
	return params, nil
}

// entered tells what g entering stage next changes beyond the library:
// Admins, the downloading Node's intent, the Import that starts and the
// library a placed file changed.
func (gr *Grabs) entered(ctx context.Context, g Grab, next Stage) {
	gr.searching.drop(g.ID)
	gr.cfg.Changed.Publish(ctx, Changed{GrabID: g.ID.String(), WorkID: g.WorkID.String()})
	switch next {
	case Downloading:
		gr.cfg.Downloads.Touch(g.Node)
	case Importing:
		gr.importer.Wake()
	case Imported:
		gr.cfg.Files.Poke()
		gr.cfg.FilesChanged.Publish(ctx, files.Changed{WorkID: g.WorkID.String()})
	case Fetching, Failed:
	}
}

// advanceParams is the update moving g by o, and o when it advances.
func advanceParams(g Grab, o outcome, now storage.Time) (db.AdvanceParams, advance) {
	params := db.AdvanceParams{ID: g.ID, Stage: g.Stage, UpdatedAt: now}
	switch o := o.(type) {
	case advance:
		params.Next = o.to
		if o.fetched != nil {
			params.FetchNodeID, params.NodeID = o.fetched.by, o.fetched.to
		}
		return params, o
	case fail:
		params.Next = Failed
		params.FailedStage, params.FailureCode, params.FailureMessage = &g.Stage, &o.code, new(o.msg)
	case dropped:
		return advanceParams(g, fail(o), now)
	}
	return params, advance{}
}

// result is how a job ended, for the schedule.
type result uint8

const (
	committed result = iota + 1
	waited
	halted
	// rerun: the step runs again at once.
	rerun
)

// step runs g's due step and commits its outcome. A wait on a Node out of
// step first awaits the roster's next change, so a Node that comes back
// resumes the step after one backoff.
func (gr *Grabs) step(ctx context.Context, g Grab) result {
	if gr.cfg.BeforeStep != nil {
		gr.cfg.BeforeStep(ctx, g.Stage)
	}
	o := steps[g.Stage](gr, ctx, g)
	switch o := o.(type) {
	case stopped:
		return halted
	case again:
		return rerun
	case wait:
		gr.cfg.Log.Warn("a Grab's step waits", "grab", g.ID.String(), "stage", string(g.Stage), "err", o.cause.Error())
		if !o.until.IsZero() {
			return gr.waitUntil(ctx, o.until)
		}
		changed := gr.cfg.Roster.Changed()
		if o.roster || o.root != nil || !gr.cfg.Roster.InStep(g.Node) {
			select {
			case <-changed:
			case <-o.root:
			case <-o.rerouted:
				return rerun
			case <-ctx.Done():
				return halted
			}
		}
		return waited
	}
	if err := gr.commit(ctx, g, o); err != nil {
		if ctx.Err() != nil {
			return halted
		}
		if _, unrouted := errors.AsType[*nodes.Unrouted](err); unrouted || errors.Is(err, downloadclients.ErrRemoving) {
			gr.cfg.Log.Warn("a Grab's step waits", "grab", g.ID.String(), "stage", string(g.Stage), "err", err.Error())
			return waited
		}
		gr.cfg.Log.Error("commit a Grab's step", "grab", g.ID.String(), "stage", string(g.Stage), "err", err.Error())
		return waited
	}
	return committed
}

// waitUntil waits until until or the Indexers' next change, so a Grab
// whose fetch waits on its Indexer's status tries again once it may.
func (gr *Grabs) waitUntil(ctx context.Context, until time.Time) result {
	t := time.NewTimer(time.Until(until))
	defer t.Stop()
	select {
	case <-t.C:
	case <-gr.cfg.Indexers.Changed():
	case <-ctx.Done():
		return halted
	}
	return rerun
}

// TorrentsOfWorks is the torrents each of ws imported from, read in r.
func TorrentsOfWorks(ctx context.Context, r storage.Reader[storage.Library], ws []works.WorkID) (map[works.WorkID][]downloadclients.ID, error) {
	rows, err := db.New(r).ListTorrentsOfWorks(ctx, ws)
	out := map[works.WorkID][]downloadclients.ID{}
	for _, row := range rows {
		out[row.WorkID] = append(out[row.WorkID], row.TorrentID)
	}
	return out, err
}
