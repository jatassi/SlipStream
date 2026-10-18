package grabs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/bencode"
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
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// outcome is what a step decided. A step does its IO outside any Write;
// commit applies its outcome in one Write.
type outcome interface{ outcome() }

// advance moves a Grab to its next stage with what that stage needs: the
// torrent and the Nodes that fetched and download it into downloading,
// the plan out of probing, the file into imported.
type advance struct {
	to      Stage
	torrent *Torrent
	fetched *fetched
	plan    *Plan
	file    *files.Incoming
}

// fetched is the Searching Node that fetched a Grab's .torrent and the
// Downloading Node that takes it.
type fetched struct{ by, to role.NodeID }

type fail struct {
	code Code
	msg  string
}

// wait retries the same step once a backoff has passed and, when the
// Grab's Node is out of step or roster is set, the roster has changed. It
// never fails the Grab.
type wait struct {
	cause  error
	roster bool
}

// stopped commits nothing: the step runs again at the next start.
type stopped struct{}

func (advance) outcome() {}
func (fail) outcome()    {}
func (wait) outcome()    {}
func (stopped) outcome() {}

// steps is the pipeline: the step each unfinished stage runs. Imported and
// Failed have none.
var steps = map[Stage]func(*Grabs, context.Context, Grab) outcome{
	Fetching:     (*Grabs).fetch,
	Downloading:  (*Grabs).downloaded,
	Probing:      (*Grabs).probe,
	Transferring: (*Grabs).transfer,
	Importing:    (*Grabs).importFile,
}

// due reports whether g has a step to run now. A downloading Grab waits
// for the Downloading Role's result, which the progress tick reads.
func (g Grab) due() bool { return !g.Done() && (g.Stage != Downloading || g.Download != nil) }

// classify turns a remote or file step's error into its outcome: a link
// that is down, mismatched or lost the call waits, and so do a Transfer's
// broken stream and a torrent the client has not re-added yet; NotHolder
// means the Node lost the Role; any other error fails the Grab with code.
func classify(ctx context.Context, err error, code Code) outcome {
	why, notSent := rpc.NotSent(err)
	switch {
	case ctx.Err() != nil:
		return stopped{}
	case notSent && why == rpc.NotHolder:
		return fail{IndexerUnavailable, err.Error()}
	case notSent, rpc.Lost(err), errors.Is(err, transfers.ErrBroken), errors.Is(err, downloading.ErrNotPresent):
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
// hands it to a Downloading Node, that one when it may download. While no
// Node may take either, it waits for the roster to change.
func (gr *Grabs) fetch(ctx context.Context, g Grab) outcome {
	if _, err := gr.cfg.Roster.Route(role.Downloading, g.Node); err != nil {
		return wait{cause: err, roster: true}
	}
	idx, err := indexers.Get(ctx, gr.cfg.Library, g.Release.Indexer)
	if err != nil {
		return local(ctx, err)
	}
	f, err := gr.cfg.Indexers.FetchTorrent(ctx, idx, g.Release.Link)
	if _, unrouted := errors.AsType[*nodes.Unrouted](err); unrouted {
		return wait{cause: err, roster: true}
	}
	if err != nil {
		return classify(ctx, err, TorrentUnreachable)
	}
	to, err := gr.cfg.Roster.Route(role.Downloading, f.Node)
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
		return fail{InfohashMismatch, fmt.Sprintf("the .torrent's info hash is %s and the Indexer's is %s", m.InfoHash, g.Release.InfoHash)}
	}
	return advance{to: Downloading, torrent: &Torrent{InfoHash: m.InfoHash, Metainfo: f.Torrent}, fetched: &fetched{by: f.Node, to: to}}
}

func (gr *Grabs) downloaded(_ context.Context, g Grab) outcome {
	if g.Download.Error != "" {
		return fail{DownloadFailed, g.Download.Error}
	}
	return advance{to: Probing}
}

func (gr *Grabs) probe(ctx context.Context, g Grab) outcome {
	video, ok := largestVideo(g.Download.Files)
	if !ok {
		return fail{NoVideoFile, "the download holds no mkv, mp4, m4v, avi or ts file"}
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
		root, err := roots.Of(ctx, r, tree.ContentType)
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
	if p.moves(g.Node) {
		return advance{to: Transferring, plan: &p}
	}
	return advance{to: Importing, plan: &p}
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

// transfer brings the file from the Node that downloaded it into the
// root's staging folder. The Transfer's id is the Grab's.
func (gr *Grabs) transfer(ctx context.Context, g Grab) outcome {
	err := gr.cfg.Transfers.Receive(ctx, transfers.Job{
		ID: transferOf(g.ID), Source: g.Node, InfoHash: g.Torrent.InfoHash, FileIndex: g.Plan.FileIndex, Staging: g.staging(),
	})
	if err != nil {
		return classify(ctx, err, TransferFailed)
	}
	return advance{to: Importing}
}

// stagingDir is the folder in a root that Transfers fill, so placing a
// transferred file is a rename on the root's filesystem.
const stagingDir = ".slipstream-staging"

func (g Grab) staging() string {
	return filepath.Join(g.Plan.Root.Path, stagingDir, g.ID.String()+".part")
}

func transferOf(id GrabID) transfers.ID { return transfers.ID{UUID: id.UUID} }

// importFile places the file in its root's holding folder: the staged
// Transfer renamed, or the local download linked. The bytes go first,
// outside any Write, under a name only this Grab uses; the commit imports
// the file, and the disk queue gives it its name.
func (gr *Grabs) importFile(ctx context.Context, g Grab) outcome {
	holding := files.HoldingOf(g.ID.UUID, path.Ext(g.Plan.Path))
	dst := filepath.Join(g.Plan.Root.Path, filepath.FromSlash(holding))
	if g.Plan.moves(g.Node) {
		if err := files.Move(g.staging(), dst); err != nil {
			return fail{ImportFailed, fmt.Sprintf("move %s: %v", holding, err)}
		}
	} else {
		src, err := gr.cfg.Payloads.Path(g.Torrent.InfoHash, g.Plan.FileIndex)
		if err != nil {
			return classify(ctx, err, ImportFailed)
		}
		if err := files.Place(gr.cfg.Linker, src, dst); err != nil {
			return fail{ImportFailed, fmt.Sprintf("place %s: %v", holding, err)}
		}
	}
	info, err := os.Stat(dst)
	if err != nil {
		return fail{ImportFailed, err.Error()}
	}
	return advance{to: Imported, file: &files.Incoming{
		WorkID: g.WorkID, ItemID: g.ItemID, RootID: g.Plan.Root.ID, Holding: holding, Size: info.Size(),
		Probe: probed(g.Plan.Probe), GrabID: g.ID.UUID, Meant: g.Targets,
	}}
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

// commit applies o to g in one Write that compares on g's stage, so a
// repeated step converges, then publishes. It is the only code that moves
// a Grab's stage.
func (gr *Grabs) commit(ctx context.Context, g Grab, o outcome) error {
	now := storage.Now()
	params, a, err := advanceParams(g, o, now)
	if err != nil {
		return err
	}
	won := false
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if n, err := q.Advance(ctx, params); err != nil || n == 0 {
			return err
		}
		if err := q.InsertStage(ctx, db.InsertStageParams{GrabID: g.ID, Stage: params.Next, EnteredAt: now}); err != nil {
			return err
		}
		for i, t := range a.fills() {
			if err := q.InsertFill(ctx, db.InsertFillParams{GrabID: g.ID, Position: int64(i), TargetID: t}); err != nil {
				return err
			}
		}
		if a.file != nil {
			if err := gr.land(ctx, tx, g, *a.file); err != nil {
				return err
			}
		}
		won = true
		return nil
	})
	if err != nil || !won {
		return err
	}
	if a.fetched != nil {
		g.Node = a.fetched.to
	}
	gr.entered(ctx, g, params.Next)
	return nil
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
// Admins, the downloading Node's intent, and its Transfer once done.
func (gr *Grabs) entered(ctx context.Context, g Grab, next Stage) {
	gr.cfg.Changed.Publish(ctx, Changed{GrabID: g.ID.String(), WorkID: g.WorkID.String()})
	if next == Downloading {
		gr.cfg.Downloads.Touch(g.Node)
	}
	if next == Imported {
		gr.cfg.Files.Poke()
		gr.cfg.FilesChanged.Publish(ctx, files.Changed{WorkID: g.WorkID.String()})
	}
	if next == Imported || next == Failed {
		if err := gr.cfg.Transfers.Forget(ctx, transferOf(g.ID)); err != nil {
			gr.cfg.Log.Error("forget a Grab's Transfer", "grab", g.ID.String(), "err", err.Error())
		}
	}
}

// advanceParams is the update moving g by o, and o when it advances.
func advanceParams(g Grab, o outcome, now storage.Time) (db.AdvanceParams, advance, error) {
	params := db.AdvanceParams{ID: g.ID, Stage: g.Stage, UpdatedAt: now}
	switch o := o.(type) {
	case advance:
		params.Next = o.to
		if o.torrent != nil {
			params.InfoHash, params.Metainfo = o.torrent.InfoHash[:], o.torrent.Metainfo
		}
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
		if o.roster || !gr.cfg.Roster.InStep(g.Node) {
			select {
			case <-changed:
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
	gr.retainTransfers(ctx)
	s := schedule{
		inflight: map[storage.UUID]bool{}, attempts: map[storage.UUID]int{}, retryAt: map[storage.UUID]time.Time{},
		shown: map[GrabID]Progress{},
	}
	done := make(chan report)
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		changed := gr.cfg.Roster.Changed()
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

// retainTransfers drops each Transfer record and staging file that no
// unfinished Grab needs, as a crash before a terminal commit's Forget
// leaves them.
func (gr *Grabs) retainTransfers(ctx context.Context) {
	_, grabs, err := gr.open(ctx)
	if err == nil {
		keep := make([]transfers.ID, len(grabs))
		for i, g := range grabs {
			keep[i] = transferOf(g.ID)
		}
		err = gr.cfg.Transfers.Retain(ctx, keep)
	}
	if err != nil && ctx.Err() == nil {
		gr.cfg.Log.Error("drop the Transfers no Grab needs", "err", err.Error())
	}
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
	if readFailed || slices.ContainsFunc(grabs, func(g Grab) bool { return g.Stage == Downloading && g.Download == nil || g.Stage == Transferring }) {
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
	for _, g := range grabs {
		p := gr.progress(g)
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
}

// progress is g's download progress from the Downloading facts, or its
// Transfer's verified bytes; nil when neither runs or reported.
func (gr *Grabs) progress(g Grab) *Progress {
	if g.Stage == Downloading {
		return gr.downloadProgress(g)
	}
	if g.Stage != Transferring {
		return nil
	}
	verified, size, ok := gr.cfg.Transfers.Progress(transferOf(g.ID))
	if !ok {
		return nil
	}
	return &Progress{Done: verified, Total: size}
}

func (gr *Grabs) downloadProgress(g Grab) *Progress {
	facts, ok := gr.cfg.Downloads.Facts(g.Node)
	if !ok {
		return nil
	}
	for _, t := range facts.GetTorrents() {
		if bencode.InfoHash(t.GetInfoHash()) == g.Torrent.InfoHash {
			return &Progress{Done: t.GetDone(), Total: t.GetWanted()}
		}
	}
	return nil
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
