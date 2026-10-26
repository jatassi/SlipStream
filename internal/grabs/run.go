package grabs

import (
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
	"github.com/jatassi/slipstream/internal/media"
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
// root is set, once it is closed or the roster has changed. It never fails
// the Grab.
type wait struct {
	cause  error
	roster bool
	root   <-chan struct{}
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
		return fail{IndexerUnavailable, err.Error()}
	case notSent, rpc.Lost(err), errors.Is(err, downloading.ErrNotPresent):
		return wait{cause: err}
	}
	return fail{code, err.Error()}
}

// local is the outcome of an error reading or writing the library inside
// a step: never a failure of the Grab.
func local(ctx context.Context, err error) outcome {
	if ctx.Err() != nil {
		return stopped{}
	}
	return wait{cause: err}
}

// fetch brings the .torrent through the Node the Indexer routes to now and
// hands it to a Downloading Node: the one holding the Work's root when it
// may download, so the file lands by a hard link. While no Node may take
// either, it waits for the roster to change.
func (gr *Grabs) fetch(ctx context.Context, g Grab) outcome {
	var idx indexers.Indexer
	var root roots.Root
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if idx, err = indexers.Get(ctx, r, g.Release.Indexer); err != nil {
			return err
		}
		root, err = roots.OfWork(ctx, r, g.WorkID)
		return err
	})
	if err != nil {
		return local(ctx, err)
	}
	if _, err := gr.cfg.Roster.Route(role.Downloading, root.Node); err != nil {
		return wait{cause: err, roster: true}
	}
	f, err := gr.cfg.Indexers.FetchTorrent(ctx, idx, g.Release.Link)
	if _, unrouted := errors.AsType[*nodes.Unrouted](err); unrouted {
		return wait{cause: err, roster: true}
	}
	if err != nil {
		return classify(ctx, err, TorrentUnreachable)
	}
	to, err := gr.cfg.Roster.Route(role.Downloading, root.Node)
	if err != nil {
		return wait{cause: err, roster: true}
	}
	m, err := bencode.DecodeMetainfo(f.Torrent)
	switch {
	case errors.Is(err, bencode.ErrV2Only):
		return fail{TorrentV2Only, err.Error()}
	case err != nil:
		return fail{TorrentInvalid, err.Error()}
	case g.Release.InfoHash != nil && *g.Release.InfoHash != m.InfoHash:
		return fail{InfohashMismatch, fmt.Sprintf("The .torrent's info hash is %s and the Indexer's is %s", m.InfoHash, g.Release.InfoHash)}
	}
	return advance{to: Downloading, torrent: &dotTorrent{metainfo: m, raw: f.Torrent}, fetched: &fetched{by: f.Node, to: to}}
}

func (gr *Grabs) downloaded(_ context.Context, g Grab) outcome {
	switch o := g.Torrent.Outcome.(type) {
	case downloadclients.Completed:
		return advance{to: Probing}
	case downloadclients.Failed:
		return fail{DownloadFailed, o.Error}
	case downloadclients.Gone:
		return failOfGone[o.Reason]
	}
	panic("grabs: a Grab whose Torrent is Pending is never due")
}

const writtenOffWords = "The Admin wrote off what this Grab's Node downloads"

// failOfGone is how a Grab at downloading fails when its Torrent is
// removed, by why.
var failOfGone = map[downloadclients.RemoveReason]fail{
	downloadclients.ByAdmin:        {Removed, "The Admin removed this Grab's torrent"},
	downloadclients.ByMarkFailed:   {DownloadFailed, "The Admin marked this Grab's download failed"},
	downloadclients.NodeWrittenOff: {NodeWrittenOff, writtenOffWords},
}

func (gr *Grabs) probe(ctx context.Context, g Grab) outcome {
	video, ok := largestVideo(g.Torrent.Files)
	if !ok {
		return fail{NoVideoFile, "The download holds no mkv, mp4, m4v, avi or ts file"}
	}
	probe, err := gr.cfg.Media.Probe(ctx, g.Node, g.Torrent.InfoHash, video.Index)
	if err != nil {
		return classify(ctx, err, ProbeFailed)
	}
	ext := path.Ext(video.Path)
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
		facts, err := gr.facts(ctx, r, g, tree.ContentType, probe)
		if err != nil {
			return err
		}
		quality, _ := facts.Quality.Get()
		placement, err := gr.cfg.Files.Place(ctx, r, files.Incoming{
			WorkID: g.WorkID, ItemID: g.ItemID, Cut: facts.Cut, RootID: root.ID, Holding: files.HoldingOf(g.ID.UUID, ext),
			Size: video.Size, Probe: probed(probe), Quality: quality, GrabID: g.ID.UUID,
			Meant: g.Targets, Unwanted: unwanted,
		})
		p, placeErr = Plan{Root: root, Path: placement.Path, Fills: placement.Fills}, err
		return nil
	})
	if err != nil {
		return local(ctx, err)
	}
	if _, nothing := errors.AsType[*files.FillsNothingError](placeErr); nothing {
		return fail{FillsNothing, placeErr.Error()}
	}
	if placeErr != nil {
		return fail{ImportFailed, placeErr.Error()}
	}
	p.FileIndex, p.Probe = video.Index, probe
	if p.Root.Node == g.Node {
		return advance{to: Importing, plan: &p}
	}
	return advance{to: Transferring, plan: &p}
}

// largestVideo is the largest file with a video extension.
func largestVideo(fs []engine.File) (engine.File, bool) {
	var best engine.File
	found := false
	for _, f := range fs {
		if media.IsVideo(f.Path) && (!found || f.Size > best.Size) {
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
	imported := advance{to: Imported, file: &files.Incoming{
		WorkID: g.WorkID, ItemID: g.ItemID, RootID: root.ID, Holding: holding.String(),
		Probe: probed(g.Plan.Probe), GrabID: g.ID.UUID, Meant: g.Targets,
	}}
	if root.Node == g.Node {
		imported.file.Size, err = gr.cfg.Roots.Place(ctx, root.ID, holding, storing.Payload{InfoHash: g.Torrent.InfoHash, FileIndex: g.Plan.FileIndex})
		if err != nil {
			return storageOutcome(ctx, err, until)
		}
		return imported
	}
	t, verified, err := gr.staged(ctx, g)
	switch {
	case err != nil:
		return local(ctx, err)
	case !verified:
		return advance{to: Transferring}
	}
	imported.file.Size, err = gr.cfg.Roots.Move(ctx, root.ID, transfers.StagedRel(t, 0), holding)
	if err != nil {
		if _, verified, rerr := gr.staged(ctx, g); rerr == nil && !verified {
			return advance{to: Transferring}
		}
		return storageOutcome(ctx, err, until)
	}
	imported.placed = &t
	return imported
}

// staged is g's Transfer, and whether it holds g's file verified at the
// place g's root is now.
func (gr *Grabs) staged(ctx context.Context, g Grab) (transfers.ID, bool, error) {
	var t transfers.Transfer
	var ok bool
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
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

// probed is what a file keeps of its probe.
func probed(p media.Probe) files.Probe {
	return files.Probe{Width: p.Width, Height: p.Height, Codec: p.Codec}
}

// facts is what g's file is, read in r: its release's source and cut,
// among the cuts a title of Content Type typ can name, and the resolution
// its probe measured.
func (gr *Grabs) facts(ctx context.Context, r storage.Reader[file], g Grab, typ works.TypeID, p media.Probe) (release.Facts, error) {
	known, err := gr.cfg.Targets.CutsIn(ctx, r)
	if err != nil {
		return release.Facts{}, err
	}
	return release.File(g.Release.Title, known.For(typ), p.Resolution()), nil
}

// errLeft rolls back a commit whose Grab has left the stage it moves
// from, along with any Torrent its attach added.
var errLeft = errors.New("the Grab left the stage")

// commit applies a step's outcome o to g in one Write that compares on
// g's stage, so a repeated step converges, then publishes. It is the only
// code that runs a step's outcome, and move the only code that writes a
// stage. A fetch's commit attaches the Grab to its Torrent, and fails with
// downloadclients.ErrRemoving while that must wait.
func (gr *Grabs) commit(ctx context.Context, g Grab, o outcome) error {
	var params db.AdvanceParams
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		o, err := confirm(ctx, tx, g, o)
		if err != nil {
			return err
		}
		params, err = gr.move(ctx, tx, g, o)
		return err
	})
	switch {
	case errors.Is(err, files.ErrRootChanged):
		return gr.commit(ctx, g, fail{RootChanged, err.Error()})
	case errors.Is(err, errLeft):
		return nil
	case err != nil:
		return err
	}
	if !params.NodeID.IsZero() {
		g.Node = params.NodeID
	}
	gr.entered(ctx, g, params.Next)
	return nil
}

// confirm re-reads a downloading Grab's Torrent in tx, so a removal that
// committed after the step read it fails the Grab rather than advancing it.
func confirm(ctx context.Context, tx *storage.Tx[file], g Grab, o outcome) (outcome, error) {
	if g.Stage != Downloading {
		return o, nil
	}
	now, err := downloadclients.Outcomes(ctx, tx, []downloadclients.ID{g.Torrent.ID})
	if err != nil {
		return nil, err
	}
	if gone, ok := now[g.Torrent.ID].Outcome.(downloadclients.Gone); ok {
		return failOfGone[gone.Reason], nil
	}
	return o, nil
}

// attach binds g to the Torrent of a's .torrent, on the Node the fetch
// picked unless another Node already holds it.
func (gr *Grabs) attach(ctx context.Context, tx *storage.Tx[file], g Grab, a advance, now storage.Time) (downloadclients.ID, role.NodeID, error) {
	tree, err := gr.cfg.Works.TreeIn(ctx, tx, g.WorkID)
	if err != nil {
		return downloadclients.ID{}, role.NodeID{}, err
	}
	return downloadclients.Attach(ctx, tx, downloadclients.Want{
		Node: a.fetched.to, Metainfo: a.torrent.metainfo, Raw: a.torrent.raw, ContentType: tree.ContentType,
		WrittenOff: func(node role.NodeID) bool { return gr.cfg.Roster.WrittenOff(role.Downloading, node) },
	}, now)
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
	facts, err := gr.facts(ctx, tx, g, tree.ContentType, g.Plan.Probe)
	if err != nil {
		return err
	}
	in.Unwanted, in.Cut = unwanted, facts.Cut
	in.Quality, _ = facts.Quality.Get()
	_, _, err = gr.cfg.Files.Import(ctx, tx, in)
	return err
}

// entered tells what g entering stage next changes beyond the library:
// Admins, the downloading Node's intent, and the Transfers move started,
// placed or cancelled.
func (gr *Grabs) entered(ctx context.Context, g Grab, next Stage) {
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
			probe, err := json.Marshal(o.plan.Probe)
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
	case wait:
		gr.cfg.Log.Warn("a Grab's step waits", "grab", g.ID.String(), "stage", string(g.Stage), "err", o.cause.Error())
		changed := gr.cfg.Roster.Changed()
		if o.roster || o.root != nil || !gr.cfg.Roster.InStep(g.Node) {
			select {
			case <-changed:
			case <-o.root:
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
		if errors.Is(err, downloadclients.ErrRemoving) {
			gr.cfg.Log.Warn("a Grab's step waits", "grab", g.ID.String(), "stage", string(g.Stage), "err", err.Error())
			return waited
		}
		gr.cfg.Log.Error("commit a Grab's step", "grab", g.ID.String(), "stage", string(g.Stage), "err", err.Error())
		return waited
	}
	return committed
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
// off, reads every open search run and unfinished Grab and starts the step
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
			for _, run := range searches {
				gr.start(ctx, &s, now, run.ID.UUID, done, func() result { return gr.search(ctx, run) })
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

// settled is the open search runs and unfinished Grabs once what the
// Admin wrote off is settled.
func (gr *Grabs) settled(ctx context.Context) ([]Search, []Grab, error) {
	searches, grabs, err := gr.open(ctx)
	if err != nil {
		return nil, nil, err
	}
	changed, err := gr.settleWriteOffs(ctx, grabs)
	if err != nil || !changed {
		return searches, grabs, err
	}
	return gr.open(ctx)
}

func (gr *Grabs) start(ctx context.Context, s *schedule, now time.Time, id storage.UUID, done chan<- report, job func() result) {
	if s.inflight[id] || now.Before(s.retryAt[id]) || ctx.Err() != nil {
		return
	}
	s.inflight[id] = true
	go func() { done <- report{id, job()} }()
}

func (s *schedule) record(r report, now time.Time, t Timings, rnd *rand.Rand) {
	switch r.result {
	case committed:
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
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: g.ID.String(), WorkID: g.WorkID.String()})
	}
}

// Progress is a download's or a Transfer's bytes done of its total.
type Progress struct {
	Done  int64
	Total int64
	// Transfer is how a transferring Grab's Transfer stands; zero while
	// it downloads.
	Transfer Pace
}

// Pace is how a Transfer stands: its state, what it waits for, and its
// rate and estimate.
type Pace struct {
	State transfers.State
	// Wait and WaitNode are zero unless State is Waiting.
	Wait     transfers.WaitKind
	WaitNode role.NodeID
	// Rate is bytes a second, 0 while unknown.
	Rate int64
	// Estimate is 0 while unknown.
	Estimate time.Duration
}

// progress is each of grabs' progress, when it has one: a downloading
// Grab's live while its Torrent is sampled, else as its Node last reported
// it, and a transferring Grab's verified bytes and pace of its Transfer.
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
			pace := Pace{State: v.State, Rate: v.Rate, Estimate: v.Estimate}
			if v.Wait != nil {
				pace.Wait, pace.WaitNode = v.Wait.Kind, v.Wait.Node
			}
			out[g.ID] = &Progress{Done: v.Verified, Total: v.Total, Transfer: pace}
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
