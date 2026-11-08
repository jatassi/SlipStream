package external

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading/internal/announce"
	"github.com/jatassi/slipstream/internal/downloading/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// state is the actor's own. Only its goroutine touches it.
type state struct {
	//nolint:containedctx // the actor's run, which Update's work runs under
	ctx   context.Context
	a     *Actor
	cfg   Config
	facts clients.Facts
	share Share
	// rows is the local rows read at the start, for torrents the share
	// lists later.
	rows map[bencode.InfoHash]db.DownloadingExternal
	// tracks is every torrent and discard of the share, by info hash.
	tracks map[bencode.InfoHash]*track

	// client is nil until a pass has the secrets for version.
	client  clients.Client
	version int64
	folders clients.Folders
	label   clients.Label
	// product is the client's version, read at its last sign-in.
	product string
	// defaultFolder is the client's default folder, read once a session.
	defaultFolder string

	reach    reach
	sentPoll *dlv1.ClientPoll
	polled   bool
	// next is when the next pass runs, zero for none until something asks.
	next     time.Time
	last     time.Time
	watchers int
	checks   []check
	totalsAt time.Time
}

type check struct {
	token []byte
	reply chan<- error
}

// track is one torrent or discard of the share: its local row and what
// the polls said of it.
type track struct {
	hash bencode.InfoHash
	id   storage.UUID
	row  db.DownloadingExternal
	// written is the row last written, nil before.
	written *db.DownloadingExternal
	intent  clients.Run
	// discard is non-nil for a discard, which parted is set on once its
	// Parted is in the outbox.
	discard *dlv1.Discard
	parted  bool
	missed  int
	// status is the last poll's, nil while the client hasn't listed it.
	status *clients.Status
	// files is its files, read once finished.
	files      []clients.File
	wanted     Wanted
	change     *WantedChange
	lastDone   int64
	lastGrowth time.Time
	// quietSince is the last poll that saw its files moving.
	quietSince time.Time
	importAt   time.Time
	unregAt    time.Time
	fault      *dlv1.Fault
	notKept    bool
	// relabeled is an added torrent listed without its label, sent once
	// more to heal it.
	relabeled  bool
	unreadable *clients.Problem
	// vanished is two successful polls without it, and told its Vanished in
	// the outbox.
	vanished, told bool
	completion     *dlv1.Completion
	sent           *dlv1.TorrentReport
	sentTotals     *dlv1.ClientCounters
}

func (t *track) live() bool { return t.discard == nil && !t.vanished }

func (t *track) origin() dlv1.Origin {
	switch {
	case t.row.Origin == nil:
		return dlv1.Origin_ORIGIN_UNSPECIFIED
	case *t.row.Origin == originAttached:
		return dlv1.Origin_ORIGIN_ATTACHED
	}
	return dlv1.Origin_ORIGIN_ADDED
}

const (
	originAdded    = "added"
	originAttached = "attached"
)

var (
	runWord   = map[clients.Run]string{clients.Started: "started", clients.Paused: "paused"}
	runOfWord = map[string]clients.Run{"started": clients.Started, "paused": clients.Paused}
)

func newState(ctx context.Context, a *Actor) *state {
	s := &state{
		ctx: ctx, a: a, cfg: a.cfg, facts: a.cfg.Kind.Facts(), tracks: map[bencode.InfoHash]*track{},
		rows: map[bencode.InfoHash]db.DownloadingExternal{},
	}
	rows, err := db.New(a.cfg.Local).ListExternal(ctx, a.cfg.ID)
	if err != nil && ctx.Err() == nil {
		s.cfg.Log.Error("read the torrents this Node holds in the client", "err", err.Error())
	}
	for _, r := range rows {
		s.rows[bencode.InfoHash(r.InfoHash)] = r
	}
	return s
}

func (s *state) update(ctx context.Context, share Share) {
	was := s.share.Client
	s.share = share
	c := share.Client
	switch {
	case was == nil || c.GetSettingsVersion() != was.GetSettingsVersion():
		s.closeClient(ctx)
		s.reach.stopped = false
		s.schedule(time.Now(), true)
	case c.GetOn() && !was.GetOn():
		s.schedule(time.Now(), true)
	case !c.GetOn():
		s.next = time.Time{}
	}
	listed := map[bencode.InfoHash]bool{}
	for _, it := range share.Torrents {
		t := s.list(it.GetId(), it.GetInfoHash())
		t.intent, t.discard = runOf(it.GetWanted()), nil
		t.wanted = t.wanted.Shown(t.intent)
		listed[t.hash] = true
	}
	for _, d := range share.Discards {
		t := s.list(d.GetId(), d.GetInfoHash())
		t.discard = d
		listed[t.hash] = true
	}
	var forget []bencode.InfoHash
	for h := range s.tracks {
		if !listed[h] {
			forget = append(forget, h)
			delete(s.tracks, h)
		}
	}
	if len(forget) > 0 {
		err := s.cfg.Local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
			for _, h := range forget {
				if err := db.New(tx).DeleteExternal(ctx, db.DeleteExternalParams{ClientID: s.cfg.ID, InfoHash: h[:]}); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil && ctx.Err() == nil {
			s.cfg.Log.Error("forget torrents the client no longer holds for SlipStream", "err", err.Error())
		}
	}
	s.publish()
}

// list is the track of hash, made from its local row when the Node has
// one.
func (s *state) list(id, hash []byte) *track {
	h := bencode.InfoHash(hash)
	t := s.tracks[h]
	if t == nil {
		t = &track{hash: h, row: db.DownloadingExternal{ClientID: s.cfg.ID, InfoHash: h[:]}}
		if r, ok := s.rows[h]; ok {
			t.row, t.written = r, &r
			delete(s.rows, h)
		}
		if t.row.Applied != nil {
			t.wanted.Applied = runOfWord[*t.row.Applied]
		}
		s.tracks[h] = t
	}
	if u, err := storage.UUIDFromBytes(id); err == nil {
		t.id = u
	}
	return t
}

func (s *state) close() { s.closeClient(context.Background()) }

func (s *state) closeClient(ctx context.Context) {
	if s.client == nil {
		return
	}
	if err := s.client.Close(ctx); err != nil {
		s.cfg.Log.Debug("close the client's session", "err", err.Error())
	}
	s.client, s.product, s.defaultFolder = nil, "", ""
}

func (s *state) watch(n int) {
	s.watchers += n
	if s.watchers == 1 && n > 0 && !s.last.IsZero() {
		s.schedule(s.last, false)
	}
}

func (s *state) check(token []byte, reply chan<- error) {
	if !s.share.Client.GetOn() {
		reply <- errors.New("the client is turned off")
		return
	}
	s.checks = append(s.checks, check{token: token, reply: reply})
	s.next = time.Now()
}

// schedule sets the next pass from last, or now when asap.
func (s *state) schedule(last time.Time, asap bool) {
	switch {
	case !s.share.Client.GetOn():
		s.next = time.Time{}
	case asap:
		s.next = last
	default:
		every := time.Duration(s.share.Client.GetCheckEverySeconds()) * time.Second
		s.next = nextPoll(last, s.reach, every, s.watchers > 0, s.cfg.Timings, s.cfg.Rand)
	}
}

// pass is one poll and everything it decides, in one local Write, then
// what it publishes. It runs at the next due time, and at once for a
// Check.
func (s *state) pass(ctx context.Context, now time.Time) {
	checks := s.checks
	s.checks = nil
	s.last = now
	defer func() { s.schedule(now, len(s.checks) > 0) }()
	if len(checks) > 0 {
		s.reach.stopped = false
	}
	if err := s.session(ctx); err != nil {
		if ctx.Err() == nil {
			s.cfg.Log.Warn("fetch the client's secrets", "err", err.Error())
		}
		answer(checks, err)
		return
	}
	b := s.poll(ctx, now)
	b.checks = checks
	if ctx.Err() != nil {
		answer(checks, ctx.Err())
		return
	}
	s.commit(ctx, b)
}

func answer(checks []check, err error) {
	for _, c := range checks {
		c.reply <- err
	}
}

// session opens the client with the secrets of its settings version,
// fetching them when the version is new.
func (s *state) session(ctx context.Context) error {
	c := s.share.Client
	if s.client != nil && s.version == c.GetSettingsVersion() {
		return nil
	}
	s.closeClient(ctx)
	call, cancel := context.WithTimeout(ctx, s.cfg.Timings.Call)
	defer cancel()
	res, err := s.cfg.Control.ClientSecrets(call, &dlv1.ClientSecretsRequest{Id: s.cfg.ID.Bytes()})
	if err != nil {
		return err
	}
	address, err := clients.ParseAddress(c.GetAddress(), s.cfg.Kind)
	if err != nil {
		return err
	}
	if s.label, err = clients.ParseLabel(c.GetLabel(), s.cfg.Kind); err != nil {
		return err
	}
	pairs := make([]clients.Folder, len(c.GetClientFolders()))
	for i, f := range c.GetClientFolders() {
		pairs[i] = clients.Folder{Client: f.GetClient(), Node: f.GetNode()}
	}
	if s.folders, err = clients.ParseFolders(pairs); err != nil {
		return err
	}
	session := clients.Session{
		HTTP: s.cfg.HTTP, Address: address, Username: c.GetUsername(),
		Secrets:      clients.Secrets{Password: res.GetSecrets().GetPassword(), APIKey: res.GetSecrets().GetApiKey()},
		TrackerEvery: s.cfg.Timings.Trackers, Call: s.cfg.Timings.Call, Log: s.cfg.Log,
	}
	if d := c.GetDaemon(); d != nil {
		session.Daemon = &clients.Daemon{Host: d.GetHost(), Port: int(d.GetPort())}
	}
	s.client, s.version = s.cfg.Open(session), c.GetSettingsVersion()
	return nil
}

// batch is what one pass writes in one local Write.
type batch struct {
	reports    []*dlv1.TorrentReport
	reported   []*track
	keys       []*dlv1.TorrentReport
	vanished   []*track
	parted     []*track
	freed      []int64
	totals     []*dlv1.ClientCounters
	totalled   []*track
	poll       *dlv1.ClientPoll
	rows       []*track
	checks     []check
	totalsSent bool
}

// poll polls the client and decides what the poll says. A failed poll
// changes nothing but the client's reach.
func (s *state) poll(ctx context.Context, now time.Time) *batch {
	b := &batch{}
	got, err := s.client.Poll(ctx, s.pollHashes())
	if err != nil {
		if ctx.Err() == nil {
			p := problemOf(err)
			s.cfg.Log.Debug("poll the client", "err", p.Error())
			s.reach = failed(s.reach, p, now, s.cfg.Timings.Grace, s.facts.Lockout)
		}
		b.poll = s.pollReport()
		return b
	}
	if v, err := s.client.Version(ctx); err == nil {
		s.product = v.String()
	}
	if s.share.Client.GetFolder() == "" && s.defaultFolder == "" {
		if f, err := s.client.DefaultFolder(ctx, s.label); err == nil {
			s.defaultFolder = f
		}
	}
	s.polled = true
	mass := s.absence(got)
	for _, t := range s.sorted() {
		switch {
		case t.discard != nil:
			s.discard(ctx, t, got, b)
		case t.vanished:
		default:
			st, present := got[t.hash]
			if present {
				s.listed(ctx, t, st, now)
			} else if !t.row.Seen {
				s.unlisted(ctx, t, now)
			}
		}
	}
	s.reach = succeeded(s.reach, s.attention(mass), now)
	s.report(now, b)
	return b
}

// pollHashes is every live torrent and pending discard of the share.
func (s *state) pollHashes() []bencode.InfoHash {
	var out []bencode.InfoHash
	for _, t := range s.sorted() {
		if t.live() || t.discard != nil && !t.parted {
			out = append(out, t.hash)
		}
	}
	return out
}

func (s *state) sorted() []*track {
	hashes := slices.SortedFunc(maps.Keys(s.tracks), func(a, b bencode.InfoHash) int { return slices.Compare(a[:], b[:]) })
	out := make([]*track, len(hashes))
	for i, h := range hashes {
		out[i] = s.tracks[h]
	}
	return out
}

// absence marks each live torrent the client listed before and then
// missed on two polls in a row vanished, and answers the mass absence's
// problem, nil for none.
func (s *state) absence(got map[bencode.InfoHash]clients.Status) *clients.Problem {
	listed := map[bencode.InfoHash]int{}
	present := map[bencode.InfoHash]bool{}
	for h, t := range s.tracks {
		if t.live() && t.row.Seen {
			listed[h] = t.missed
			_, present[h] = got[h]
		}
	}
	vanished, missed, mass := absence(listed, present, s.cfg.Timings.MissingShare, s.cfg.Timings.MissingFloor)
	if mass {
		absent := 0
		for _, ok := range present {
			if !ok {
				absent++
			}
		}
		return &clients.Problem{Cause: clients.MostMissing, Missing: absent}
	}
	for h, n := range missed {
		s.tracks[h].missed = n
	}
	for _, h := range vanished {
		s.tracks[h].vanished = true
	}
	return nil
}

// listed is one live torrent the poll lists.
func (s *state) listed(ctx context.Context, t *track, st clients.Status, now time.Time) {
	if !t.row.Seen && !s.firstSight(ctx, t, st, now) {
		return
	}
	t.status = &st
	if st.Done > t.lastDone || t.lastGrowth.IsZero() {
		t.lastDone, t.lastGrowth = st.Done, now
	}
	if st.Activity == clients.Moving {
		t.quietSince = now
	}
	if *t.row.Origin == originAdded && st.Label != s.label.String() && !t.relabeled {
		t.relabeled = s.send(ctx, t)
	}
	s.wantedSync(ctx, t, st)
	s.trackers(t, st, now)
	s.finish(ctx, t, st, now)
}

// firstSight records a torrent the client lists for the first time:
// attached when the Node didn't add it, by the Controller's record when
// the Node lost its own. It waits for a later poll while the Controller
// can't say.
func (s *state) firstSight(ctx context.Context, t *track, st clients.Status, now time.Time) bool {
	if t.row.Origin == nil {
		res, err := s.fetch(ctx, t)
		if err != nil {
			return false
		}
		origin := originAttached
		if res.GetOrigin() == dlv1.Origin_ORIGIN_ADDED {
			origin = originAdded
		}
		t.row.Origin = &origin
	}
	t.row.Seen, t.row.SentAt, t.row.Resends, t.notKept = true, storage.Time{}, 0, false
	if t.row.Applied == nil {
		t.wanted.Applied = st.Run
	}
	t.lastGrowth = now
	return true
}

func (s *state) fetch(ctx context.Context, t *track) (*dlv1.TorrentResponse, error) {
	call, cancel := context.WithTimeout(ctx, s.cfg.Timings.Call)
	defer cancel()
	res, err := s.cfg.Control.Torrent(call, &dlv1.TorrentRequest{Id: t.id.Bytes()})
	if err != nil && ctx.Err() == nil {
		s.cfg.Log.Warn("fetch a torrent's metainfo", "info_hash", t.hash.String(), "err", err.Error())
	}
	return res, err
}

// unlisted is one live torrent the client has never listed: added once,
// sent once more after the resend wait, then not kept.
func (s *state) unlisted(ctx context.Context, t *track, now time.Time) {
	if t.row.Origin == nil {
		if s.share.MayStart {
			s.add(ctx, t, now)
		}
		return
	}
	if *t.row.Origin != originAdded {
		return
	}
	switch resend(t.row.SentAt.Time(), int(t.row.Resends), now, s.cfg.Timings.ResendWait) {
	case ResendWait:
	case ResendNow:
		if s.add(ctx, t, now) {
			t.row.Resends++
		}
	case ResendNotKept:
		t.notKept = true
	}
}

// add sends the torrent and records it as added.
func (s *state) add(ctx context.Context, t *track, now time.Time) bool {
	if !s.send(ctx, t) {
		return false
	}
	added := originAdded
	t.row.Origin, t.row.SentAt = &added, storage.FromTime(now)
	t.wanted.Applied = t.intent
	return true
}

// send sends the torrent's .torrent bytes to its Content Type's subfolder
// of the download folder with the label, paused when the intent wants it
// paused.
func (s *state) send(ctx context.Context, t *track) bool {
	res, err := s.fetch(ctx, t)
	if err != nil || res.GetOrigin() == dlv1.Origin_ORIGIN_ATTACHED {
		return false
	}
	base := s.share.Client.GetFolder()
	if base == "" {
		if base, err = s.client.DefaultFolder(ctx, s.label); err != nil {
			s.cfg.Log.Warn("read the client's default folder", "err", err.Error())
			return false
		}
		s.defaultFolder = base
	}
	err = s.client.Add(ctx, clients.Add{
		Metainfo: res.GetMetainfo(), Hash: t.hash, Folder: joinClient(base, s.cfg.Subfolder(res.GetContentType())),
		Label: s.label, Paused: t.intent == clients.Paused,
	})
	if err != nil {
		s.cfg.Log.Warn("add a torrent to the client", "info_hash", t.hash.String(), "err", err.Error())
		return false
	}
	return true
}

// joinClient is name under folder, with the separator the client's own
// path uses.
func joinClient(folder, name string) string {
	sep := "/"
	if slices.Contains([]byte(folder), '\\') {
		sep = `\`
	}
	for len(folder) > 1 && (folder[len(folder)-1] == '/' || folder[len(folder)-1] == '\\') {
		folder = folder[:len(folder)-1]
	}
	return folder + sep + name
}

func (s *state) wantedSync(ctx context.Context, t *track, st clients.Status) {
	send, change, next := wantedStep(t.intent, st.Run, st.StoppedAtLimit, t.wanted)
	if send != 0 {
		act := map[clients.Run]clients.Act{clients.Started: clients.Resume, clients.Paused: clients.Pause}[send]
		if err := s.client.Act(ctx, act, []bencode.InfoHash{t.hash}); err != nil {
			s.cfg.Log.Warn("pause or resume a torrent in the client", "info_hash", t.hash.String(), "err", err.Error())
			return
		}
		t.status.Run = send
	}
	if change != nil {
		t.change = change
	}
	t.wanted = next
}

// trackers raises the unregistered release fault once a tracker has
// called the torrent unregistered for the unregistered gap.
func (s *state) trackers(t *track, st clients.Status, now time.Time) {
	if !slices.ContainsFunc(st.Trackers, func(m clients.TrackerMessage) bool { return m.Error && announce.Unregistered(m.Message) }) {
		t.unregAt = time.Time{}
		return
	}
	if t.unregAt.IsZero() {
		t.unregAt = now
	}
	if t.fault == nil && unregistered(t.unregAt, now, s.share.UnregisteredGap) {
		t.fault = &dlv1.Fault{Kind: dlv1.FaultKind_FAULT_KIND_UNREGISTERED, AtUnixMs: now.UnixMilli()}
	}
}

// finish holds a finished torrent's Completion back for the Import delay,
// then sends it once every file opens on this Node through the Client
// folders.
func (s *state) finish(ctx context.Context, t *track, st clients.Status, now time.Time) {
	if !st.Finished {
		t.importAt = time.Time{}
		return
	}
	if t.row.FinishedAt.IsZero() {
		t.row.FinishedAt = storage.FromTime(now)
	}
	if t.row.Completed {
		if t.files == nil {
			t.files, _ = s.files(ctx, t)
		}
		return
	}
	delay := time.Duration(s.share.Client.GetImportDelaySeconds()) * time.Second
	t.importAt = importAt(st.FinishedAt, t.row.FinishedAt.Time(), t.quietSince, delay)
	if now.Before(t.importAt) {
		return
	}
	files, err := s.files(ctx, t)
	if err != nil {
		return
	}
	t.files, t.unreadable = files, nil
	completion := &dlv1.Completion{}
	for _, f := range files {
		path, used, ok := s.folders.OnNode(joinClient(st.Folder, f.Path), filepath.Separator)
		if !ok {
			t.unreadable = &clients.Problem{Cause: clients.FolderUnreadable, Folder: st.Folder}
			return
		}
		file, err := os.Open(path) //nolint:gosec // a torrent's file as the client's Client folders place it
		if err != nil {
			t.unreadable = &clients.Problem{Cause: clients.FolderUnreadable, Folder: filepath.Dir(path), Paired: used.Client}
			return
		}
		_ = file.Close()
		completion.Files = append(completion.Files, &dlv1.File{Index: int32(f.Index), Path: f.Path, Size: f.Size}) //nolint:gosec // a torrent's file index fits
	}
	t.row.Completed, t.completion = true, completion
}

func (s *state) files(ctx context.Context, t *track) ([]clients.File, error) {
	files, err := s.client.Files(ctx, t.hash)
	if err != nil && ctx.Err() == nil {
		s.cfg.Log.Warn("list a torrent's files in the client", "info_hash", t.hash.String(), "err", err.Error())
	}
	return files, err
}

// discard confirms a removed torrent: an attached one, or one whose origin
// the Node lost, only stops being tracked and the client is never called.
// One the Node added is removed from the client, with its files when the
// discard says so.
func (s *state) discard(ctx context.Context, t *track, got map[bencode.InfoHash]clients.Status, b *batch) {
	if t.parted {
		return
	}
	var freed int64
	if _, present := got[t.hash]; present && t.origin() == dlv1.Origin_ORIGIN_ADDED {
		var ok bool
		if freed, ok = s.remove(ctx, t); !ok {
			return
		}
	}
	b.parted, b.freed = append(b.parted, t), append(b.freed, freed)
}

func (s *state) remove(ctx context.Context, t *track) (int64, bool) {
	del := t.discard.GetDeleteFiles()
	var freed int64
	var paths []string
	if del {
		c := s.copyOf(ctx, t.id)
		freed, paths = s.a.frees(c).GetBytes(), c.paths
	}
	if err := s.client.Remove(ctx, t.hash, del && s.facts.DeletesData); err != nil {
		s.cfg.Log.Warn("remove a torrent from the client", "info_hash", t.hash.String(), "err", err.Error())
		return 0, false
	}
	if del && !s.facts.DeletesData {
		s.deleteFiles(paths)
	}
	return freed, true
}

// deleteFiles deletes a removed torrent's files on this Node, for a kind
// whose remove never deletes them, and the folders they leave empty.
func (s *state) deleteFiles(files []string) {
	within, ok := s.a.Folder()
	if !ok {
		s.cfg.Log.Warn("delete a removed torrent's files: the client's download folder on this Node isn't known yet")
		return
	}
	paths, dirs, err := deletable(files, within)
	if err != nil {
		s.cfg.Log.Warn("delete a removed torrent's files", "err", err.Error())
		return
	}
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			s.cfg.Log.Warn("delete a removed torrent's file", "err", err.Error())
		}
	}
	for _, d := range dirs {
		_ = os.Remove(d)
	}
}

// attention is the problem a successful poll leaves the client in, most
// pressing first.
func (s *state) attention(mass *clients.Problem) *clients.Problem {
	if mass != nil {
		return mass
	}
	var notKept *clients.Problem
	for _, t := range s.sorted() {
		if !t.live() {
			continue
		}
		if t.unreadable != nil {
			return t.unreadable
		}
		if t.notKept && notKept == nil {
			notKept = &clients.Problem{Cause: clients.NotKept}
		}
	}
	return notKept
}

// copyOf is torrent id's files as this Node sees them, read from the
// client while the actor holds none.
func (s *state) copyOf(ctx context.Context, id storage.UUID) fileCopy {
	c := fileCopy{id: id}
	t := s.byID(id)
	if t == nil {
		return c
	}
	c.attached = t.origin() == dlv1.Origin_ORIGIN_ATTACHED
	if t.files == nil && t.status != nil && s.client != nil {
		t.files, _ = s.files(ctx, t)
	}
	if t.status == nil {
		return c
	}
	for _, f := range t.files {
		path, _, ok := s.folders.OnNode(joinClient(t.status.Folder, f.Path), filepath.Separator)
		if !ok {
			continue
		}
		c.paths, c.done = append(c.paths, path), append(c.done, f.Done)
	}
	return c
}

func (s *state) byID(id storage.UUID) *track {
	for _, t := range s.tracks {
		if t.id == id {
			return t
		}
	}
	return nil
}
