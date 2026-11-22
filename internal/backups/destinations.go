package backups

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/transfers"
	tsource "github.com/jatassi/slipstream/internal/transfers/source"
)

// DestinationID is a Backup destination's own id.
type DestinationID [16]byte

func (d DestinationID) String() string { return hex.EncodeToString(d[:]) }

func (d DestinationID) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

func (d *DestinationID) UnmarshalText(b []byte) error {
	if hex.DecodedLen(len(b)) != len(d) {
		return fmt.Errorf("a destination id is %d hex characters, not %d", hex.EncodedLen(len(d)), len(b))
	}
	if _, err := hex.Decode(d[:], b); err != nil {
		return fmt.Errorf("read a destination id: %w", err)
	}
	return nil
}

// ParseDestinationID reads an id as String writes it.
func ParseDestinationID(s string) (DestinationID, error) {
	var d DestinationID
	err := d.UnmarshalText([]byte(s))
	return d, err
}

// Destination is a folder on a Node that holds a copy of every Backup.
// It needs no Role. Changing one is removing it and adding another.
type Destination struct {
	ID   DestinationID `json:"id"`
	Node role.NodeID   `json:"node"`
	// Folder is as the Admin typed it, with either separator.
	Folder  string    `json:"folder"`
	AddedAt time.Time `json:"addedAt"`
	// RemovedAt is set once the Admin removed it. It stays listed, with
	// nothing new queued to it, until no copy Transfer still runs into its
	// folder, so one in flight finishes there.
	RemovedAt time.Time `json:"removedAt,omitzero"`
}

type CopyState string

const (
	CopyWaiting CopyState = "waiting"
	// Copying has a Transfer, whose outcome is read from the transfers
	// ledger.
	Copying    CopyState = "copying"
	Copied     CopyState = "copied"
	CopyFailed CopyState = "failed"
)

// Copy is a Backup's copy on one destination. Its state is the
// destination's word: the Node's Backup folder facts and its answers to
// checks.
type Copy struct {
	Destination DestinationID `json:"destination"`
	State       CopyState     `json:"state"`
	CopiedAt    time.Time     `json:"copiedAt,omitzero"`
	// Cause is a failed copy's, in words.
	Cause string `json:"cause,omitempty"`
	Check Check  `json:"check,omitzero"`
}

type CheckResult string

const (
	// CheckAsked is a check the destination hasn't answered yet.
	CheckAsked CheckResult = "asked"
	CheckOK    CheckResult = "ok"
	// CheckMismatch is a keyed BLAKE3 that isn't the Backup's.
	CheckMismatch CheckResult = "mismatch"
	// CheckUnreadable is a destination that couldn't read its copy.
	CheckUnreadable CheckResult = "unreadable"
	// CheckNoLocalFile is a Backup with no file on this machine to check
	// its copy against.
	CheckNoLocalFile CheckResult = "no_local_file"
)

// Check is the last check of a copy: a fresh key the destination's Node
// answers a keyed BLAKE3 of its copy under, compared with the Backup's
// own.
type Check struct {
	Key     Sum         `json:"key,omitzero"`
	AskedAt time.Time   `json:"askedAt,omitzero"`
	At      time.Time   `json:"at,omitzero"`
	Result  CheckResult `json:"result,omitempty"`
	Cause   string      `json:"cause,omitempty"`
}

// Pull is a restore's pull of a Backup from a destination back into the
// Backups folder. Once it arrives and its BLAKE3 matches the index, the
// Backup is Local and the pull goes.
type Pull struct {
	From  DestinationID `json:"from"`
	State CopyState     `json:"state"`
	Cause string        `json:"cause,omitempty"`
}

var (
	ErrNoDestination     = errors.New("no Backup destination has that id")
	ErrDestinationExists = errors.New("that folder is already a Backup destination")
	ErrNoCopy            = errors.New("that destination has no copy of the Backup")
	ErrNoBackup          = errors.New("no Backup has that id")
)

// UnusableError is a folder its Node refused as a Backup destination.
type UnusableError struct{ Refusal roots.Refusal }

func (e *UnusableError) Error() string {
	return "the folder can't be a Backup destination: " + string(e.Refusal.Reason)
}

// Ledger is the index as it is now.
func (b *Backups) Ledger() Ledger { return b.cfg.Engine.Index().Read() }

// Destination is the destination with id, removed or not.
func (l *Ledger) Destination(id DestinationID) (Destination, bool) {
	i := slices.IndexFunc(l.Destinations, func(d Destination) bool { return d.ID == id })
	if i < 0 {
		return Destination{}, false
	}
	return l.Destinations[i], true
}

// Active is every destination the Admin hasn't removed.
func (l *Ledger) Active() []Destination {
	return slices.DeleteFunc(slices.Clone(l.Destinations), func(d Destination) bool { return !d.RemovedAt.IsZero() })
}

// Copy is e's copy on d, false for none.
func (e *Entry) Copy(d DestinationID) (Copy, bool) {
	i := slices.IndexFunc(e.Copies, func(c Copy) bool { return c.Destination == d })
	if i < 0 {
		return Copy{}, false
	}
	return e.Copies[i], true
}

func (e *Entry) setCopy(c Copy) {
	if i := slices.IndexFunc(e.Copies, func(o Copy) bool { return o.Destination == c.Destination }); i >= 0 {
		e.Copies[i] = c
		return
	}
	e.Copies = append(e.Copies, c)
}

func (e *Entry) dropCopy(d DestinationID) {
	e.Copies = slices.DeleteFunc(e.Copies, func(c Copy) bool { return c.Destination == d })
}

// Added is a destination just added, and the room its Node lacks to keep
// every Backup listed now: zero when it has enough.
type Added struct {
	Destination Destination
	Need, Free  int64
}

// Add checks folder on node with the backups purpose and adds it as a
// Backup destination, then queues a copy of every Backup to it. Its
// Node's verdict refuses with *UnusableError. A folder the Node can't
// hold every listed Backup in is added anyway, with Need and Free set.
func (b *Backups) Add(ctx context.Context, admin identity.Admin, node role.NodeID, folder string) (Added, error) {
	v, err := b.cfg.Check(ctx, admin, node, folder)
	if err != nil {
		return Added{}, err
	}
	if v.Refusal != nil {
		return Added{}, &UnusableError{Refusal: *v.Refusal}
	}
	d := Destination{Node: node, Folder: folder, AddedAt: b.cfg.Engine.cfg.Clock().UTC()}
	if _, err := io.ReadFull(b.cfg.Engine.cfg.Rand, d.ID[:]); err != nil {
		return Added{}, fmt.Errorf("draw a destination id: %w", err)
	}
	var need int64
	err = b.cfg.Engine.Index().Update(func(l *Ledger) error {
		if slices.ContainsFunc(l.Active(), func(o Destination) bool { return o.Node == node && o.Folder == folder }) {
			return ErrDestinationExists
		}
		l.Destinations = append(l.Destinations, d)
		for _, e := range l.Backups {
			if e.State == Verified {
				need += e.Size
			}
		}
		return nil
	})
	if err != nil {
		return Added{}, err
	}
	b.touchFolders()
	if err := b.startDestinations(ctx, []Destination{d}); err != nil {
		return Added{}, err
	}
	out := Added{Destination: d}
	if v.Volume != nil && need > v.Volume.Free {
		out.Need, out.Free = need, v.Volume.Free
	}
	return out, nil
}

// Remove removes destination id. It deletes and cancels nothing: its
// copies stay on its Node, a copy in flight finishes into its folder, and
// that outcome changes nothing.
func (b *Backups) Remove(_ context.Context, id DestinationID) error {
	b.copyMu.Lock()
	defer b.copyMu.Unlock()
	err := b.cfg.Engine.Index().Update(func(l *Ledger) error {
		i := slices.IndexFunc(l.Destinations, func(d Destination) bool { return d.ID == id && d.RemovedAt.IsZero() })
		if i < 0 {
			return ErrNoDestination
		}
		l.Destinations[i].RemovedAt = b.cfg.Engine.cfg.Clock().UTC()
		for j := range l.Backups {
			l.Backups[j].dropCopy(id)
		}
		return nil
	})
	if err != nil {
		return err
	}
	b.touchFolders()
	b.Poke()
	return nil
}

// Folders is transfers.ControllerConfig.Folders: node's Backup folders,
// read from the index alone. A destination's folder lists the names of
// the Backups being deleted that it still holds and the check asked of
// its newest copy. The Controller's own Backups folder is listed for it,
// so it sends copies and receives pulls.
func (b *Backups) Folders(node role.NodeID) []transfers.BackupFolder {
	l := b.cfg.Engine.Index().Read()
	var out []transfers.BackupFolder
	if node == b.cfg.Self() {
		out = append(out, transfers.BackupFolder{Folder: b.cfg.Engine.cfg.Folder})
	}
	for _, d := range l.Destinations {
		if d.Node != node {
			continue
		}
		f := transfers.BackupFolder{Folder: d.Folder}
		for _, e := range l.Backups {
			c, ok := e.Copy(d.ID)
			switch {
			case !ok:
			case e.State == Deleting:
				f.Delete = append(f.Delete, e.Name.String())
			case c.Check.Result == CheckAsked:
				f.CheckKey, f.Check = c.Check.Key[:], e.Name.String()
			}
		}
		out = append(out, f)
	}
	return out
}

// FoldersNoted is transfers.ControllerConfig.FoldersNoted.
func (b *Backups) FoldersNoted(role.NodeID) { b.Poke() }

// Poke asks Run to settle the copies again. It never blocks.
func (b *Backups) Poke() {
	select {
	case b.poke <- struct{}{}:
	default:
	}
}

// Run settles the copies whenever a copy Transfer moves, a Node reports
// its Backup folders or the index changes, and publishes backups.changed
// after each index change, until ctx ends.
func (b *Backups) Run(ctx context.Context) error {
	for {
		if err := b.settle(ctx); err != nil && ctx.Err() == nil {
			b.cfg.Log.Error("settle the Backup copies", "err", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-b.poke:
		case <-b.cfg.Engine.Index().Changed():
			b.cfg.Changed.Publish(ctx, Changed{})
			b.touchFolders()
		}
	}
}

// touchFolders tells every Node with a Backup folder that its folders may
// have changed.
func (b *Backups) touchFolders() {
	l := b.cfg.Engine.Index().Read()
	ids := []role.NodeID{b.cfg.Self()}
	for _, d := range l.Destinations {
		if !slices.Contains(ids, d.Node) {
			ids = append(ids, d.Node)
		}
	}
	b.cfg.Transfers.TouchFolders(ids...)
}

// job keys a Backup's Transfer: "copy/<destination>/<backup>" or
// "pull/<backup>".
type job struct {
	pull   bool
	dest   DestinationID
	backup archive.ID
}

func (j job) String() string {
	if j.pull {
		return "pull/" + hex.EncodeToString(j.backup[:])
	}
	return "copy/" + j.dest.String() + "/" + hex.EncodeToString(j.backup[:])
}

func parseJob(s string) (job, bool) {
	parts := strings.Split(s, "/")
	var j job
	var backup string
	switch {
	case len(parts) == 2 && parts[0] == "pull":
		j.pull, backup = true, parts[1]
	case len(parts) == 3 && parts[0] == "copy":
		if err := j.dest.UnmarshalText([]byte(parts[1])); err != nil {
			return job{}, false
		}
		backup = parts[2]
	default:
		return job{}, false
	}
	if err := j.backup.UnmarshalText([]byte(backup)); err != nil {
		return job{}, false
	}
	return j, true
}

// purpose registers the Backup Transfers. Their hooks write nothing: Run
// reads each outcome from the transfers ledger, then ends the record, so
// a crash between leaves the outcome for the next settle.
func (b *Backups) purpose() transfers.Purpose {
	nothing := func(context.Context, *storage.Tx[storage.Library], transfers.Transfer, string) error { return nil }
	return transfers.Purpose{
		Kind:     transfers.Backup,
		Verified: nothing,
		Failed: func(context.Context, *storage.Tx[storage.Library], transfers.Transfer, transfers.Failure) error {
			return nil
		},
		After: func([]string) { b.Poke() },
		Names: func(_ context.Context, _ storage.Reader[storage.Library], jobs []string) (map[string]string, error) {
			l := b.cfg.Engine.Index().Read()
			out := map[string]string{}
			for _, s := range jobs {
				if j, ok := parseJob(s); ok {
					if e, ok := l.Entry(j.backup); ok {
						out[s] = e.Name.String()
					}
				}
			}
			return out, nil
		},
	}
}

// destinationUnit is the Backup task's unit for d.
func destinationUnit(d DestinationID) tasks.UnitKey {
	return tasks.UnitKey("destination/" + d.String())
}

func destinationOf(u tasks.UnitKey) (DestinationID, bool) {
	s, ok := strings.CutPrefix(string(u), "destination/")
	if !ok {
		return DestinationID{}, false
	}
	d, err := ParseDestinationID(s)
	return d, err == nil
}

// units is the Backup task's units: the Backup, then one per destination
// tied to its Node, so a Node that was away catches up when it returns.
func (b *Backups) units(ctx context.Context, r storage.Reader[storage.Library]) ([]tasks.Unit, error) {
	peers, err := nodes.PeersIn(ctx, r)
	if err != nil {
		return nil, err
	}
	out := []tasks.Unit{{Key: backupUnit, Label: "Library", Place: tasks.OnController()}}
	l := b.cfg.Engine.Index().Read()
	for _, d := range l.Active() {
		out = append(out, tasks.Unit{Key: destinationUnit(d.ID), Label: d.Folder + " on " + peers[d.Node].Name, Place: tasks.TiedTo(d.Node)})
	}
	return out, nil
}

// startDestinations starts the units of ds ByEvent.
func (b *Backups) startDestinations(ctx context.Context, ds []Destination) error {
	if len(ds) == 0 {
		return nil
	}
	keys := make([]tasks.UnitKey, len(ds))
	for i, d := range ds {
		keys[i] = destinationUnit(d.ID)
	}
	_, err := b.cfg.Start(ctx, tasks.Start{Task: TaskID, Units: keys, Trigger: tasks.ByEvent})
	return err
}

// queueCopies is a destination unit's run: it queues a copy of every verified
// Backup on this machine that the destination lacks, Low, and asks a
// check of its newest copy when one is due.
func (b *Backups) queueCopies(ctx context.Context, id DestinationID, p *tasks.Progress) error {
	b.copyMu.Lock()
	defer b.copyMu.Unlock()
	l := b.cfg.Engine.Index().Read()
	d, ok := l.Destination(id)
	if !ok || !d.RemovedAt.IsZero() {
		return nil
	}
	every, queued, err := b.want(ctx, l, d)
	if err != nil {
		return err
	}
	now := b.cfg.Engine.cfg.Clock().UTC()
	var key Sum
	if _, err := io.ReadFull(b.cfg.Engine.cfg.Rand, key[:]); err != nil {
		return fmt.Errorf("draw a check key: %w", err)
	}
	err = b.cfg.Engine.Index().Update(func(l *Ledger) error {
		for i := range l.Backups {
			e := &l.Backups[i]
			if slices.Contains(queued, e.ID) {
				c, _ := e.Copy(id)
				c.Destination, c.State, c.Cause = id, Copying, ""
				e.setCopy(c)
			}
		}
		askCheck(l, id, every, key, now)
		return nil
	})
	if err != nil {
		return err
	}
	b.cfg.Transfers.Touch()
	b.touchFolders()
	if len(queued) > 0 {
		p.Note(fmt.Sprintf("Queued %d %s to copy.", len(queued), noun.Of(int64(len(queued)))))
	}
	return nil
}

// want Wants a copy to d of every verified Backup on this machine that d
// has no copy of in flight or done, and reads Check copies.
func (b *Backups) want(ctx context.Context, l Ledger, d Destination) (time.Duration, []archive.ID, error) {
	var every time.Duration
	var queued []archive.ID
	err := b.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		if every, err = checkCopiesIn(ctx, tx); err != nil {
			return err
		}
		queued = nil
		for _, e := range l.Backups {
			c, has := e.Copy(d.ID)
			if e.State != Verified || !e.Local || has && (c.State == Copying || c.State == Copied) {
				continue
			}
			if _, err := b.cfg.Transfers.Want(ctx, tx, b.copyOf(e, d)); err != nil {
				return err
			}
			queued = append(queued, e.ID)
		}
		return nil
	})
	return every, queued, err
}

// askCheck asks a check of destination id's newest copied Backup once
// every has passed since its last check, or since it was copied. A Backup
// with no file here can't be checked, and the check says so.
func askCheck(l *Ledger, id DestinationID, every time.Duration, key Sum, now time.Time) {
	for i := range l.Backups {
		e := &l.Backups[i]
		c, ok := e.Copy(id)
		if !ok || c.State != Copied || e.State != Verified {
			continue
		}
		last := c.Check.At
		if last.IsZero() {
			last = c.CopiedAt
		}
		if c.Check.Result == CheckAsked || now.Before(last.Add(every)) {
			return
		}
		if !e.Local {
			c.Check = Check{At: now, Result: CheckNoLocalFile}
		} else {
			c.Check = Check{Key: key, AskedAt: now, Result: CheckAsked}
		}
		e.setCopy(c)
		return
	}
}

// copyOf is the Request that copies e to d.
func (b *Backups) copyOf(e Entry, d Destination) transfers.Request {
	return transfers.Request{
		Purpose: transfers.Backup, Job: job{dest: d.ID, backup: e.ID}.String(), Source: b.cfg.Self(), Priority: transfers.Low,
		Destination: transfers.BackupDestination{ID: d.ID, Node: d.Node, Folder: d.Folder},
		Files:       []transfers.FileRequest{{Name: e.Name.String(), Source: tsource.Backup{Folder: b.cfg.Engine.cfg.Folder, Name: e.Name.String()}, Size: e.Size}},
	}
}

// Pull starts pulling Backup id back from destination from into the
// Backups folder, for a restore of a Backup this machine doesn't hold.
// Once it arrives and its BLAKE3 matches the index, the Backup is Local
// at Path; one that doesn't match is deleted and the pull fails.
func (b *Backups) Pull(ctx context.Context, id archive.ID, from DestinationID) error {
	b.copyMu.Lock()
	defer b.copyMu.Unlock()
	l := b.cfg.Engine.Index().Read()
	e, ok := l.Entry(id)
	if !ok {
		return ErrNoBackup
	}
	d, ok := l.Destination(from)
	if c, has := e.Copy(from); !ok || !has || c.State != Copied {
		return ErrNoCopy
	}
	req := transfers.Request{
		Purpose: transfers.Backup, Job: job{pull: true, backup: id}.String(), Source: d.Node,
		Destination: transfers.BackupsFolder{Node: b.cfg.Self(), Folder: b.cfg.Engine.cfg.Folder},
		Files:       []transfers.FileRequest{{Name: e.Name.String(), Source: tsource.Backup{Folder: d.Folder, Name: e.Name.String()}, Size: e.Size}},
	}
	if err := b.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := b.cfg.Transfers.Want(ctx, tx, req)
		return err
	}); err != nil {
		return err
	}
	if err := b.cfg.Engine.Index().Update(func(l *Ledger) error {
		e, ok := l.Entry(id)
		if !ok {
			return ErrNoBackup
		}
		e.Pull = &Pull{From: from, State: Copying}
		l.set(e)
		return nil
	}); err != nil {
		return err
	}
	b.cfg.Transfers.Touch()
	b.touchFolders()
	return nil
}

// outcome is one Backup Transfer's record as settle reads it.
type outcome struct {
	t    transfers.Transfer
	done bool
}

// settle brings the index to what the transfers ledger and the Nodes'
// Backup folders say, then ends the records it read an outcome from:
//
//   - a copy whose Transfer verified is copied, and one whose Transfer
//     failed is failed with its words; a copying copy whose Transfer the
//     library doesn't know waits again;
//   - an outcome for a destination removed since changes nothing;
//   - a copied copy its Node no longer lists is failed;
//   - a check answered under its key is compared with the Backup's own
//     keyed BLAKE3, and a mismatch fails the copy and queues it again;
//   - a copy of a Backup being deleted goes once its Node no longer lists
//     it, and the Backup leaves the index once its file and every copy
//     are gone;
//   - a pulled Backup that arrived whose BLAKE3 is the index's is Local;
//   - a removed destination leaves the index once no Transfer runs into
//     it.
func (b *Backups) settle(ctx context.Context) error {
	requeue, err := b.settled(ctx)
	if err != nil {
		return err
	}
	for _, d := range requeue {
		if err := b.queueCopies(ctx, d.ID, &tasks.Progress{}); err != nil {
			return err
		}
	}
	return nil
}

// settled is settle's index and ledger writes, and the destinations whose
// copies it failed on a check.
func (b *Backups) settled(ctx context.Context) ([]Destination, error) {
	b.copyMu.Lock()
	defer b.copyMu.Unlock()
	st := settling{b: b}
	if err := b.cfg.Library.Read(ctx, st.read); err != nil {
		return nil, err
	}
	l := b.cfg.Engine.Index().Read()
	st.sums, st.keyed, st.now = b.pulledSums(l, st.records), b.keyedSums(l), b.cfg.Engine.cfg.Clock().UTC()
	var ended []transfers.Transfer
	err := b.cfg.Engine.Index().Update(func(l *Ledger) error {
		st.requeue = nil
		ended = st.outcomes(l)
		for i := range l.Backups {
			st.follow(l, &l.Backups[i])
		}
		l.Backups = slices.DeleteFunc(l.Backups, func(e Entry) bool { return e.State == Deleting && !e.Local && len(e.Copies) == 0 })
		l.Destinations = slices.DeleteFunc(l.Destinations, func(d Destination) bool {
			return !d.RemovedAt.IsZero() && !inFlight(st.records, d.ID)
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return st.requeue, b.end(ctx, ended)
}

// settling is what one settle read: the Backup Transfers by job, the
// Nodes' names, the BLAKE3 of each pulled Backup and of each check out.
type settling struct {
	b       *Backups
	records map[string]outcome
	names   map[role.NodeID]string
	sums    map[archive.ID]*Sum
	keyed   map[keyedAt]Sum
	now     time.Time
	requeue []Destination
}

func (st *settling) read(ctx context.Context, rd storage.Reader[storage.Library]) error {
	all, err := st.b.cfg.Transfers.OfPurpose(ctx, rd, transfers.Backup)
	if err != nil {
		return err
	}
	st.records = make(map[string]outcome, len(all))
	for _, t := range all {
		st.records[t.Job] = outcome{t: t, done: len(t.Files) > 0 && !slices.ContainsFunc(t.Files, func(f transfers.File) bool { return !f.Verified })}
	}
	peers, err := nodes.PeersIn(ctx, rd)
	if err != nil {
		return err
	}
	st.names = make(map[role.NodeID]string, len(peers))
	for id, p := range peers {
		st.names[id] = p.Name
	}
	return nil
}

// outcomes writes each ended Transfer's outcome into l and returns those
// Transfers. One for a destination removed since changes nothing.
func (st *settling) outcomes(l *Ledger) []transfers.Transfer {
	var ended []transfers.Transfer
	for s, o := range st.records {
		j, ok := parseJob(s)
		if !ok || !o.done && o.t.Failure == nil {
			continue
		}
		ended = append(ended, o.t)
		e, ok := l.Entry(j.backup)
		if !ok {
			continue
		}
		if j.pull {
			pulled(&e, o, st.sums[e.ID])
			l.set(e)
			continue
		}
		c, ok := e.Copy(j.dest)
		if d, known := l.Destination(j.dest); !ok || !known || !d.RemovedAt.IsZero() {
			continue
		}
		if o.done {
			c.State, c.CopiedAt, c.Cause, c.Check = Copied, st.now, "", Check{}
		} else {
			c.State, c.Cause = CopyFailed, o.t.Failure.Sentence
		}
		e.setCopy(c)
		l.set(e)
	}
	return ended
}

// follow brings e's pull and copies to what the ledger and each Node's
// facts say.
func (st *settling) follow(l *Ledger, e *Entry) {
	if e.Pull != nil && e.Pull.State == Copying {
		if _, ok := st.records[job{pull: true, backup: e.ID}.String()]; !ok {
			e.Pull = nil
		}
	}
	for _, c := range slices.Clone(e.Copies) {
		d, _ := l.Destination(c.Destination)
		if c.State == Copying {
			if _, ok := st.records[job{dest: c.Destination, backup: e.ID}.String()]; !ok {
				c.State = CopyWaiting
			}
		}
		facts, ok := st.b.folderFacts(d)
		read := ok && facts.Error == ""
		gone := read && !slices.Contains(facts.Names, e.Name.String())
		switch {
		case e.State == Deleting && gone:
			e.dropCopy(c.Destination)
			continue
		case c.State == Copied && gone:
			c.State, c.Cause = CopyFailed, fmt.Sprintf("%s no longer has this copy in %s.", st.names[d.Node], d.Folder)
		case c.Check.Result == CheckAsked && ok && answers(facts.Check, e.Name, c.Check.Key):
			st.checked(e, &c, d, facts.Check)
		}
		e.setCopy(c)
	}
}

// answers is whether a is the answer to the check of name under key.
func answers(a *transfers.CheckAnswer, name archive.Name, key Sum) bool {
	return a != nil && bytes.Equal(a.Key, key[:]) && a.Name == name.String()
}

// checked compares d's answer to the check of e's copy c with the
// Backup's own keyed BLAKE3. A mismatch fails the copy and queues it
// again.
func (st *settling) checked(e *Entry, c *Copy, d Destination, a *transfers.CheckAnswer) {
	c.Check.At = st.now
	switch want, known := st.keyed[keyedAt{e.ID, c.Check.Key}]; {
	case a.Error != "":
		c.Check.Result, c.Check.Cause = CheckUnreadable, st.names[d.Node]+" couldn't read its copy of the Backup."
		st.b.cfg.Log.Warn("check a Backup copy", "node", st.names[d.Node], "folder", d.Folder, "backup", e.Name, "err", a.Error)
	case known && bytes.Equal(a.Sum, want[:]):
		c.Check.Result, c.Check.Cause = CheckOK, ""
	default:
		c.Check.Result, c.Check.Cause = CheckMismatch, ""
		c.State, c.Cause = CopyFailed, fmt.Sprintf("The copy on %s doesn't match the Backup.", st.names[d.Node])
		if !slices.Contains(st.requeue, d) {
			st.requeue = append(st.requeue, d)
		}
	}
}

// inFlight is whether a copy Transfer still runs into destination d.
func inFlight(records map[string]outcome, d DestinationID) bool {
	for s, o := range records {
		if j, ok := parseJob(s); ok && !j.pull && j.dest == d && !o.done && o.t.Failure == nil {
			return true
		}
	}
	return false
}

// pulled is e once its pull's Transfer ended: Local when the file that
// arrived has the index's BLAKE3, else the pull failed and the file is
// gone.
func pulled(e *Entry, o outcome, sum *Sum) {
	if e.Pull == nil {
		return
	}
	switch {
	case o.t.Failure != nil:
		e.Pull.State, e.Pull.Cause = CopyFailed, o.t.Failure.Sentence
	case sum != nil && *sum == e.Sum:
		e.Local, e.Pull = true, nil
	default:
		e.Pull.State, e.Pull.Cause = CopyFailed, "The copy doesn't match the Backup, so SlipStream deleted it."
	}
}

// pulledSums reads the BLAKE3 of each pulled Backup that arrived, and
// deletes one that doesn't match the index.
func (b *Backups) pulledSums(l Ledger, records map[string]outcome) map[archive.ID]*Sum {
	out := map[archive.ID]*Sum{}
	for _, e := range l.Backups {
		o, ok := records[job{pull: true, backup: e.ID}.String()]
		if !ok || !o.done {
			continue
		}
		path := b.cfg.Engine.path(e.Name)
		sum, err := fileSum(path, nil)
		if err == nil && sum == e.Sum {
			out[e.ID] = &sum
			continue
		}
		if err := b.cfg.Engine.remove(path); err != nil {
			b.cfg.Log.Warn("delete a pulled copy that doesn't match", "backup", e.Name, "err", err)
		}
	}
	return out
}

type keyedAt struct {
	backup archive.ID
	key    Sum
}

// keyedSums is the keyed BLAKE3 of each local Backup a check is out for,
// under that check's key.
func (b *Backups) keyedSums(l Ledger) map[keyedAt]Sum {
	out := map[keyedAt]Sum{}
	for _, e := range l.Backups {
		for _, c := range e.Copies {
			at := keyedAt{e.ID, c.Check.Key}
			if c.Check.Result != CheckAsked || !e.Local {
				continue
			}
			if _, done := out[at]; done {
				continue
			}
			sum, err := fileSum(b.cfg.Engine.path(e.Name), c.Check.Key[:])
			if err != nil {
				b.cfg.Log.Warn("read a Backup to check its copy", "backup", e.Name, "err", err)
				continue
			}
			out[at] = sum
		}
	}
	return out
}

// fileSum is the BLAKE3 of the file at path, keyed when key is set.
func fileSum(path string, key []byte) (Sum, error) {
	f, err := os.Open(path) //nolint:gosec // path is a Backup name in the Backups folder
	if err != nil {
		return Sum{}, fmt.Errorf("read the Backup: %w", err)
	}
	defer func() { _ = f.Close() }()
	h := blake3.New()
	if key != nil {
		if h, err = blake3.NewKeyed(key); err != nil {
			return Sum{}, fmt.Errorf("start the check: %w", err)
		}
	}
	if _, err := io.Copy(h, f); err != nil {
		return Sum{}, fmt.Errorf("read the Backup: %w", err)
	}
	var s Sum
	copy(s[:], h.Sum(nil))
	return s, nil
}

// folderFacts is d's folder as its Node last reported it.
func (b *Backups) folderFacts(d Destination) (transfers.FolderFacts, bool) {
	for _, f := range b.cfg.Transfers.FoldersOf(d.Node) {
		if f.Folder == d.Folder {
			return f, true
		}
	}
	return transfers.FolderFacts{}, false
}

// end ends each Backup Transfer settle read an outcome from: a verified
// one is placed, since its receiver renamed it into place, and a failed
// one goes, so the next queue starts it again.
func (b *Backups) end(ctx context.Context, ended []transfers.Transfer) error {
	if len(ended) == 0 {
		return nil
	}
	err := b.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, t := range ended {
			if t.Failure != nil {
				if _, err := b.cfg.Transfers.Want(ctx, tx, transfers.Request{Purpose: transfers.Backup, Job: t.Job}); err != nil {
					return err
				}
				continue
			}
			for _, f := range t.Files {
				if err := b.cfg.Transfers.Placed(ctx, tx, t.ID, f.Position); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	b.cfg.Transfers.Touch()
	return nil
}
