package grabs

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"path"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// outcome is what a step decided. A step does its IO outside any Write;
// commit applies its outcome in one Write.
type outcome interface{ outcome() }

// advance moves a Grab to its next stage with what that stage needs: the
// .torrent and the Nodes that fetched and download it into downloading,
// the plan out of probing, the file into imported and, when a Transfer
// staged it, that Transfer.
type advance struct {
	to      Stage
	torrent *dotTorrent
	fetched *fetched
	plan    *Plan
	file    *files.Incoming
	placed  *transfers.ID
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

// steps is the pipeline: the step each stage runs. A transferring Grab
// has none: its Transfer's hooks move it. Imported and Failed have none.
var steps = map[Stage]func(*Grabs, context.Context, Grab) outcome{
	Fetching:    (*Grabs).fetch,
	Downloading: (*Grabs).downloaded,
	Probing:     (*Grabs).probe,
	Importing:   (*Grabs).importFile,
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
		return fail{InfohashMismatch, fmt.Sprintf("The .torrent's info hash is %s and the Indexer's is %s", m.InfoHash, g.Release.InfoHash)}
	}
	return advance{to: Downloading, torrent: &dotTorrent{metainfo: m, raw: raw}, fetched: &f}
}

func (gr *Grabs) downloaded(_ context.Context, g Grab) outcome {
	switch o := g.Torrent.Outcome.(type) {
	case downloadclients.Completed:
		return advance{to: Probing}
	case downloadclients.Failed:
		return fail{DownloadFailed, o.Error}
	case downloadclients.Gone:
		return failOfGone(o)
	}
	panic("grabs: a Grab whose Torrent is Pending is never due")
}

const writtenOffWords = "The Admin wrote off what this Grab's Node downloads"

// failOfGone is how a Grab fails when its Torrent is removed, by why. The
// two client reasons name the client.
func failOfGone(g downloadclients.Gone) fail {
	switch g.Reason {
	case downloadclients.ByAdmin:
		return fail{Removed, "The Admin removed this Grab's torrent"}
	case downloadclients.ByMarkFailed:
		if g.By == "" {
			return fail{MarkedFailed, "Marked failed through the API key."}
		}
		return fail{MarkedFailed, g.By + " marked its download failed."}
	case downloadclients.NodeWrittenOff:
		return fail{NodeWrittenOff, writtenOffWords}
	case downloadclients.ByClient:
		return fail{Removed, g.Client + " removed the torrent before SlipStream imported it."}
	case downloadclients.ClientDeleted:
		return fail{Removed, "The Admin deleted " + g.Client + " before SlipStream imported this torrent."}
	case downloadclients.GoalMet:
		return fail{Removed, "SlipStream removed this Grab's torrent once it met its seeding goal."}
	case downloadclients.ReleaseFaulted:
		return failOfFault(g)
	case downloadclients.Canceled:
		return fail{Canceled, "This Grab's torrent was removed when another Grab of it was canceled."}
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

func (gr *Grabs) probe(ctx context.Context, g Grab) outcome {
	video, ok := largestVideo(g.Torrent.Files)
	if !ok {
		return fail{NoVideoFile, "The download holds no video file."}
	}
	record, err := gr.cfg.Media.Probe(ctx, g.Node, g.Torrent.InfoHash, video.Index)
	if err != nil {
		return classify(ctx, err, ProbeFailed)
	}
	holding := files.HoldingOf(g.ID.UUID, path.Ext(video.Path))
	var p Plan
	var placeErr error
	err = gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		tree, err := gr.cfg.Works.TreeIn(ctx, r, g.WorkID)
		if err != nil {
			return err
		}
		root, err := roots.OfWork(ctx, r, g.WorkID)
		if err != nil {
			return err
		}
		unwanted, err := db.New(r).GetUnwanted(ctx, g.ID)
		if err != nil {
			return err
		}
		read, facts, err := gr.facts(ctx, r, g, tree.ContentType, record)
		if err != nil {
			return err
		}
		quality, _ := facts.Quality.Get()
		placement, err := gr.cfg.Files.Place(ctx, r, files.Incoming{
			WorkID: g.WorkID, ItemID: g.ItemID, Cut: facts.Cut, RootID: root.ID, Holding: holding,
			Size: video.Size, Media: record, Quality: quality, ReleaseName: g.Release.Title, Title: read, SourceName: path.Base(video.Path),
			GrabID: g.ID.UUID, Meant: g.Targets, Unwanted: unwanted,
		})
		p, placeErr = Plan{Root: root, Path: placement.Path, Fills: placement.Fills}, err
		return nil
	})
	if err != nil {
		return local(ctx, err)
	}
	if f, ok := refused(placeErr); ok {
		return f
	}
	if placeErr != nil {
		return fail{ImportFailed, placeErr.Error()}
	}
	until := gr.cfg.Roots.Watch()
	if err := gr.cfg.Files.Vacant(ctx, p.Root.ID, holding, p.Path); err != nil {
		if f, ok := refused(err); ok {
			return f
		}
		return storageOutcome(ctx, err, until)
	}
	p.FileIndex, p.Media = video.Index, record
	if p.Root.Node == g.Node {
		return advance{to: Importing, plan: &p}
	}
	return advance{to: Transferring, plan: &p}
}

// refused is the failure of a Grab whose file the library refuses: it
// fills nothing, its name is taken, or its name doesn't fit on its root.
func refused(err error) (fail, bool) {
	var nothing *files.FillsNothingError
	var taken *files.NameTakenError
	var long *naming.TooLongError
	switch {
	case errors.As(err, &nothing):
		return fail{FillsNothing, nothing.Error()}, true
	case errors.As(err, &taken):
		return fail{NameTaken, taken.Error()}, true
	case errors.As(err, &long):
		return fail{NameTooLong, long.Error()}, true
	}
	return fail{}, false
}

// largestVideo is the largest file that counts as a video, so a sample
// or a bonus feature is never the one imported.
func largestVideo(fs []engine.File) (engine.File, bool) {
	var best engine.File
	found := false
	for _, f := range fs {
		if media.Counts(f.Path) && (!found || f.Size > best.Size) {
			best, found = f, true
		}
	}
	return best, found
}

// importFile places the file in its root's holding folder, on the Node
// that holds the root: the download linked when that Node downloaded it,
// else the file its Transfer staged renamed. The root is read as it is
// now, so a re-home never strands a Grab on a stale plan. The bytes go
// first, outside any Write, under a name only this Grab uses; the commit
// imports the file, and the disk queue gives it its name. A root that
// isn't ready makes it wait for the root to change, and a staged file not
// verified where the root is now sends the Grab back to transferring.
func (gr *Grabs) importFile(ctx context.Context, g Grab) outcome {
	root := g.Plan.Root
	holding, err := storing.ParseRel(files.HoldingOf(g.ID.UUID, path.Ext(g.Plan.Path)))
	if err != nil {
		return fail{ImportFailed, err.Error()}
	}
	until := gr.cfg.Roots.Watch()
	i := slices.IndexFunc(g.Torrent.Files, func(f engine.File) bool { return f.Index == g.Plan.FileIndex })
	imported := advance{to: Imported, file: &files.Incoming{
		WorkID: g.WorkID, ItemID: g.ItemID, RootID: root.ID, Holding: holding.String(),
		Media: g.Plan.Media, SourceName: path.Base(g.Torrent.Files[i].Path), GrabID: g.ID.UUID, Meant: g.Targets,
	}}
	if root.Node == g.Node {
		placed, err := gr.cfg.Roots.Place(ctx, root.ID, holding, storing.Payload{InfoHash: g.Torrent.InfoHash, FileIndex: g.Plan.FileIndex})
		if err != nil {
			return storageOutcome(ctx, err, until)
		}
		imported.file.Size, imported.file.Seen = placed.Size, placed.Seen
		return imported
	}
	t, verified, err := gr.staged(ctx, g)
	switch {
	case err != nil:
		return local(ctx, err)
	case !verified:
		return advance{to: Transferring}
	}
	moved, err := gr.cfg.Roots.Move(ctx, root.ID, transfers.StagedRel(t, 0), holding, false)
	if err != nil {
		if _, verified, rerr := gr.staged(ctx, g); rerr == nil && !verified {
			return advance{to: Transferring}
		}
		return storageOutcome(ctx, err, until)
	}
	imported.file.Size, imported.file.Seen = moved.Size, moved.Seen
	imported.placed = &t
	return imported
}

// staged is g's Transfer, and whether it holds g's file verified at the
// place g's root is now.
func (gr *Grabs) staged(ctx context.Context, g Grab) (id transfers.ID, verified bool, err error) {
	var t transfers.Transfer
	var ok bool
	err = gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		t, ok, err = gr.cfg.Transfers.Find(ctx, r, transfers.Grab, g.ID.String())
		return err
	})
	if err != nil || !ok || t.Failure != nil {
		return transfers.ID{}, false, err
	}
	return t.ID, !slices.ContainsFunc(t.Files, func(f transfers.File) bool { return !f.Verified && !f.Placed }), nil
}

// storageOutcome turns a root's refusal or a Storage call's error into its
// outcome. A root that may not take the bytes now, or a Node that lost
// Storage, waits for until, a root's next change; a Node the call didn't
// reach waits for the roster, and a call lost on the way, or a download
// its Node hasn't finished re-adding, waits a backoff. Another file at the
// target, or any other error, fails the Grab.
func storageOutcome(ctx context.Context, err error, until <-chan struct{}) outcome {
	why, notSent := rpc.NotSent(err)
	_, notReady := errors.AsType[*storing.NotReady](err)
	switch {
	case ctx.Err() != nil:
		return stopped{}
	case notReady, notSent && why == rpc.NotHolder:
		return wait{cause: err, root: until}
	case notSent:
		return wait{cause: err, roster: true}
	case rpc.Lost(err):
		return wait{cause: err}
	}
	return fail{ImportFailed, err.Error()}
}

// facts is what g's file is, read in r: its release title read by
// Content Type typ, among the cuts a title of typ can name, and that
// reading's facts at the resolution its media record measured. A Grab an
// Admin grabbed as another Edition has that Edition's cut, so its file is
// stored under it and every later refill keeps the fill.
func (gr *Grabs) facts(ctx context.Context, r storage.Reader[file], g Grab, typ works.TypeID, m media.Record) (release.Read, release.Facts, error) {
	known, err := gr.cfg.Targets.CutsIn(ctx, r)
	if err != nil {
		return release.Read{}, release.Facts{}, err
	}
	read := release.Title(g.Release.Title, gr.cfg.Names[typ], known.For(typ))
	facts := read.Facts.WithProbe(release.Probed{Resolution: release.Known(m.Resolution())})
	if g.Overrides.Edition != "" {
		facts.Cut = release.CutKey(g.Edition)
	}
	return read, facts, nil
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
	var taken *Grab
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
	f, isRefused := refused(err)
	switch {
	case errors.Is(err, files.ErrRootChanged):
		return gr.commit(ctx, g, fail{RootChanged, err.Error()})
	case errors.Is(err, errBlocked):
		return gr.commit(ctx, g, dropped{Blocklisted, blockedWords})
	case isRefused:
		return gr.commit(ctx, g, f)
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
	if taken != nil {
		gr.cfg.Indexers.Taken(ctx, taken.Ticket)
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: taken.ID.String(), WorkID: taken.WorkID.String()})
	}
	return nil
}

// confirm refuses a fetch whose .torrent the Blocklist holds with
// errBlocked, before anything reaches a client. It re-reads a Grab's
// Torrent in tx, so a removal that committed after the step read it fails
// the Grab by its reason: a downloading Grab rather than advancing it, and
// a later one rather than failing it for the files the removal took.
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
	_, failing := o.(fail)
	if g.Stage == Fetching || g.Stage != Downloading && !failing {
		return o, nil
	}
	now, err := downloadclients.Outcomes(ctx, tx, []downloadclients.ID{g.Torrent.ID})
	if err != nil {
		return nil, err
	}
	gone, ok := now[g.Torrent.ID].Outcome.(downloadclients.Gone)
	if ok && (g.Stage == Downloading || slices.Contains(failsAtAnyStage, gone.Reason)) {
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

// move writes o to g in tx, or fails with errLeft when g has left the
// stage it stood at when read. It is the only code that writes a Grab's
// stage: commit calls it for a step's outcome, and the transfers hooks
// inside the Write that verified or failed the Grab's Transfer. In the
// same Write, a move into transferring starts the Grab's Transfer, one
// into imported places the file it staged, and one into imported or
// failed cancels whatever Transfer the Grab still has.
func (gr *Grabs) move(ctx context.Context, tx *storage.Tx[file], g Grab, o outcome) (db.AdvanceParams, error) {
	now := storage.Now()
	params, a, err := advanceParams(g, o, now)
	if err != nil {
		return params, err
	}
	if a.torrent != nil {
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
	for i, t := range a.fills() {
		if err := q.InsertFill(ctx, db.InsertFillParams{GrabID: g.ID, Position: int64(i), TargetID: t}); err != nil {
			return params, err
		}
	}
	if a.file != nil {
		if err := gr.land(ctx, tx, g, *a.file); err != nil {
			return params, err
		}
	}
	if a.placed != nil {
		if err := gr.cfg.Transfers.Placed(ctx, tx, *a.placed, 0); err != nil {
			return params, err
		}
	}
	switch params.Next {
	case Transferring:
		if a.plan != nil {
			g.Plan = a.plan
		}
		return params, gr.startTransfer(ctx, tx, g)
	case Imported, Failed:
		return params, gr.cancelTransfer(ctx, tx, g.ID)
	case Fetching, Downloading, Probing, Importing:
	}
	return params, nil
}

// land imports g's file in tx, with its facts, under the choice a change
// that took its cells saved, if one did.
func (gr *Grabs) land(ctx context.Context, tx *storage.Tx[file], g Grab, in files.Incoming) error {
	unwanted, err := db.New(tx).GetUnwanted(ctx, g.ID)
	if err != nil {
		return err
	}
	tree, err := gr.cfg.Works.TreeIn(ctx, tx, g.WorkID)
	if err != nil {
		return err
	}
	read, facts, err := gr.facts(ctx, tx, g, tree.ContentType, g.Plan.Media)
	if err != nil {
		return err
	}
	in.Unwanted, in.Cut, in.ReleaseName, in.Title = unwanted, facts.Cut, g.Release.Title, read
	in.Quality, _ = facts.Quality.Get()
	_, _, err = gr.cfg.Files.Import(ctx, tx, in)
	return err
}

// entered tells what g entering stage next changes beyond the library:
// Admins, the downloading Node's intent, and the Transfers move started,
// placed or canceled.
func (gr *Grabs) entered(ctx context.Context, g Grab, next Stage) {
	gr.searching.drop(g.ID)
	gr.cfg.Changed.Publish(ctx, Changed{GrabID: g.ID.String(), WorkID: g.WorkID.String()})
	switch next {
	case Downloading:
		gr.cfg.Downloads.Touch(g.Node)
	case Imported:
		gr.cfg.Files.Poke()
		gr.cfg.FilesChanged.Publish(ctx, files.Changed{WorkID: g.WorkID.String()})
		gr.cfg.Transfers.Touch()
	case Transferring, Failed:
		gr.cfg.Transfers.Touch()
	case Fetching, Probing, Importing:
	}
}

// advanceParams is the update moving g by o, and o when it advances.
func advanceParams(g Grab, o outcome, now storage.Time) (db.AdvanceParams, advance, error) {
	params := db.AdvanceParams{ID: g.ID, Stage: g.Stage, UpdatedAt: now}
	switch o := o.(type) {
	case advance:
		params.Next = o.to
		if o.fetched != nil {
			params.FetchNodeID, params.NodeID = o.fetched.by, o.fetched.to
		}
		if o.plan != nil {
			probe, err := json.Marshal(o.plan.Media)
			if err != nil {
				return db.AdvanceParams{}, advance{}, fmt.Errorf("encode the probe: %w", err)
			}
			params.FileIndex = new(int64(o.plan.FileIndex))
			params.Probe, params.RootID, params.Path = new(string(probe)), o.plan.Root.ID, new(o.plan.Path)
		}
		return params, o, nil
	case fail:
		params.Next = Failed
		params.FailedStage, params.FailureCode, params.FailureMessage = &g.Stage, &o.code, new(o.msg)
	case dropped:
		return advanceParams(g, fail(o), now)
	}
	return params, advance{}, nil
}

func (a advance) fills() []targets.TargetID {
	if a.plan == nil {
		return nil
	}
	return a.plan.Fills
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

type report struct {
	id     storage.UUID
	result result
}

// schedule is the reconciler's memory: what runs, and when a waiting job
// may run again. Only Run's goroutine touches it.
type schedule struct {
	inflight map[storage.UUID]bool
	attempts map[storage.UUID]int
	retryAt  map[storage.UUID]time.Time
	shown    map[GrabID]Progress
	ticked   time.Time
}

// Run reconciles until ctx ends. Each pass settles what the Admin wrote
// off and what a client removed, reads every open search run and unfinished Grab and starts the step
// each is due, unless it runs or backs off. A pass runs at each change of
// the roster too. A restart needs no recovery: the first pass is the
// recovery.
func (gr *Grabs) Run(ctx context.Context) error {
	s := schedule{
		inflight: map[storage.UUID]bool{}, attempts: map[storage.UUID]int{}, retryAt: map[storage.UUID]time.Time{},
		shown: map[GrabID]Progress{},
	}
	done := make(chan report)
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		changed, moved := gr.cfg.Roster.Changed(), gr.cfg.Downloads.Moved()
		searches, grabs, err := gr.settled(ctx)
		now := time.Now()
		if err == nil {
			s.keep(searches, grabs)
			for _, run := range searches {
				gr.start(ctx, &s, now, run.ID.UUID, done, gr.runJob(ctx, run))
			}
			for _, g := range grabs {
				if g.due() {
					gr.start(ctx, &s, now, g.ID.UUID, done, func() result { return gr.step(ctx, g) })
				}
			}
			gr.publishProgress(ctx, &s, now, grabs)
		} else if ctx.Err() == nil {
			gr.cfg.Log.Error("read or settle the open Grabs", "err", err.Error())
		}
		timer.Reset(s.next(now, grabs, err != nil, gr.cfg.Timings))
		select {
		case r := <-done:
			delete(s.inflight, r.id)
			s.record(r, time.Now(), gr.cfg.Timings, gr.cfg.Rand)
		case <-gr.poke:
		case <-changed:
		case <-moved:
		case <-timer.C:
		case <-ctx.Done():
			for range s.inflight {
				<-done
			}
			return nil
		}
	}
}

// settled is the open search runs and unfinished Grabs once the release
// faults, what the Admin wrote off and what a client removed are settled.
func (gr *Grabs) settled(ctx context.Context) ([]Search, []Grab, error) {
	if err := gr.settleFaults(ctx); err != nil {
		return nil, nil, err
	}
	searches, grabs, err := gr.open(ctx)
	if err != nil {
		return nil, nil, err
	}
	changed, err := gr.settleGone(ctx, grabs)
	if err != nil || !changed {
		return searches, grabs, err
	}
	return gr.open(ctx)
}

// runJob is run's job: an automatic run searches and decides, and a
// Manual run listens.
func (gr *Grabs) runJob(ctx context.Context, run Search) func() result {
	if run.Manual.IsZero() {
		return func() result { return gr.search(ctx, run) }
	}
	return func() result { return gr.listen(ctx, run) }
}

func (gr *Grabs) start(ctx context.Context, s *schedule, now time.Time, id storage.UUID, done chan<- report, job func() result) {
	if s.inflight[id] || now.Before(s.retryAt[id]) || ctx.Err() != nil {
		return
	}
	s.inflight[id] = true
	go func() { done <- report{id, job()} }()
}

// keep forgets each job neither open nor running, as one whose Work was
// deleted: a vanished job ends as a finished one does, and its past retry
// never wakes the reconciler again.
func (s *schedule) keep(searches []Search, grabs []Grab) {
	open := make(map[storage.UUID]bool, len(searches)+len(grabs))
	for _, run := range searches {
		open[run.ID.UUID] = true
	}
	for _, g := range grabs {
		open[g.ID.UUID] = true
	}
	for id := range s.retryAt {
		if !open[id] && !s.inflight[id] {
			delete(s.retryAt, id)
			delete(s.attempts, id)
		}
	}
	for id := range s.shown {
		if !open[id.UUID] {
			delete(s.shown, id)
		}
	}
}

func (s *schedule) record(r report, now time.Time, t Timings, rnd *rand.Rand) {
	switch r.result {
	case committed, rerun:
		delete(s.attempts, r.id)
		delete(s.retryAt, r.id)
	case waited:
		s.retryAt[r.id] = now.Add(backoff.Delay(t.RetryMin, t.RetryMax, s.attempts[r.id], rnd))
		s.attempts[r.id]++
	case halted:
	}
}

// next is how long the reconciler sleeps: until the earliest retry of a
// job not running, as a running one may be waiting on the roster, and
// no longer than one progress tick while a Grab downloads or transfers or
// a read failed.
func (s *schedule) next(now time.Time, grabs []Grab, readFailed bool, t Timings) time.Duration {
	wake := now.Add(time.Hour)
	if readFailed || slices.ContainsFunc(grabs, func(g Grab) bool { return g.waitsOnTorrent() || g.Stage == Transferring }) {
		wake = now.Add(t.Progress)
	}
	for id, at := range s.retryAt {
		if !s.inflight[id] && at.Before(wake) {
			wake = at
		}
	}
	return max(wake.Sub(now), 0)
}

// publishProgress sends grabs.changed for each downloading or transferring
// Grab whose progress moved, at most once per progress tick.
func (gr *Grabs) publishProgress(ctx context.Context, s *schedule, now time.Time, grabs []Grab) {
	if now.Sub(s.ticked) < gr.cfg.Timings.Progress {
		return
	}
	s.ticked = now
	progress, err := gr.progress(ctx, grabs)
	if err != nil {
		if ctx.Err() == nil {
			gr.cfg.Log.Warn("read the Grabs' Transfers", "err", err.Error())
		}
		return
	}
	for _, g := range grabs {
		p := progress[g.ID]
		if p == nil || s.shown[g.ID] == *p {
			continue
		}
		s.shown[g.ID] = *p
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: g.ID.String(), WorkID: g.WorkID.String(), Progress: true})
	}
}

// Progress is a download's or a Transfer's bytes done of its total.
type Progress struct {
	Done  int64
	Total int64
}

// progress is each of grabs' progress, when it has one: a downloading
// Grab's live while its Torrent is sampled, else as its Node last reported
// it, and a transferring Grab's verified bytes of its Transfer.
func (gr *Grabs) progress(ctx context.Context, grabs []Grab) (map[GrabID]*Progress, error) {
	out := map[GrabID]*Progress{}
	var jobs []string
	for _, g := range grabs {
		switch g.Stage {
		case Downloading:
			if p := gr.downloadProgress(g); p != nil {
				out[g.ID] = p
			}
		case Transferring:
			jobs = append(jobs, g.ID.String())
		case Fetching, Probing, Importing, Imported, Failed:
		}
	}
	if len(jobs) == 0 {
		return out, nil
	}
	views, err := gr.cfg.Transfers.ViewsOf(ctx, transfers.Grab, jobs)
	if err != nil {
		return nil, err
	}
	for _, g := range grabs {
		if v, ok := views[g.ID.String()]; ok {
			out[g.ID] = &Progress{Done: v.Verified, Total: v.Total}
		}
	}
	return out, nil
}

func (gr *Grabs) downloadProgress(g Grab) *Progress {
	if s, ok := gr.cfg.Downloads.Sample(g.Torrent.ID); ok {
		return &Progress{Done: s.Done, Total: s.Wanted}
	}
	p := g.Torrent.Progress
	if p == nil {
		return nil
	}
	return &Progress{Done: p.Done, Total: p.Size}
}

// open reads, in one snapshot, every open search run and unfinished Grab,
// oldest first.
func (gr *Grabs) open(ctx context.Context) ([]Search, []Grab, error) {
	var searches []Search
	var grabs []Grab
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		runs, err := q.ListOpenSearches(ctx)
		if err != nil {
			return err
		}
		for _, run := range runs {
			s, err := searchOf(run, nil)
			if err != nil {
				return err
			}
			searches = append(searches, s)
		}
		rows, err := q.ListOpenGrabs(ctx)
		if err != nil {
			return err
		}
		grabs, err = loadAll(ctx, r, rows)
		return err
	})
	return searches, grabs, err
}

func (gr *Grabs) wake() {
	select {
	case gr.poke <- struct{}{}:
	default:
	}
}
