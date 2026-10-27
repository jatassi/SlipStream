package indexers

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"
	"unicode"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// Privacy is whether a Tracker admits anyone.
type Privacy string

const (
	Private     Privacy = "private"
	SemiPrivate Privacy = "semi-private"
	Public      Privacy = "public"
	Unknown     Privacy = "unknown"
)

// Tracker is a site that Indexers reach.
type Tracker struct {
	ID      TrackerID
	Name    string
	Privacy Privacy
	// Hosts are the announce hosts of its private torrents.
	Hosts []string
	// Keys are how a Release finds it: "definition:<id>",
	// "prowlarr:<feed Indexer>:<source id>" or "named:<Indexer>".
	Keys []string
	// Indexers are the Indexers that reach it, by name.
	Indexers []ID
}

var (
	ErrNoTracker        = errors.New("no such Tracker")
	ErrTrackerNameTaken = errors.New("another Tracker has that name")
	ErrMergeSelf        = errors.New("a Tracker can't be merged into itself")
)

// Repoint is how the owners of rows naming a Tracker move them in a
// merge's Write: grabs and downloadclients register one each.
type Repoint func(ctx context.Context, tx *storage.Tx[storage.Library], from, into TrackerID) error

// Trackers is every Tracker by name.
func (x *Indexers) Trackers(ctx context.Context, _ identity.Admin) ([]Tracker, error) {
	var out []Tracker
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		out, err = trackersIn(ctx, r)
		return err
	})
	return out, err
}

// Tracker is Tracker id, or ErrNoTracker.
func (x *Indexers) Tracker(ctx context.Context, by identity.Admin, id TrackerID) (Tracker, error) {
	all, err := x.Trackers(ctx, by)
	if err != nil {
		return Tracker{}, err
	}
	i := slices.IndexFunc(all, func(t Tracker) bool { return t.ID == id })
	if i < 0 {
		return Tracker{}, ErrNoTracker
	}
	return all[i], nil
}

// RenameTracker names Tracker id name. Its keys stay, so every Release
// that reached it still does. It fails with ErrNoTracker or
// ErrTrackerNameTaken.
func (x *Indexers) RenameTracker(ctx context.Context, _ identity.Admin, id TrackerID, name string) error {
	err := x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		taken, err := q.TrackerNameTaken(ctx, db.TrackerNameTakenParams{Name: name, Except: id})
		if err != nil {
			return err
		}
		if taken {
			return ErrTrackerNameTaken
		}
		n, err := q.RenameTracker(ctx, db.RenameTrackerParams{Name: name, ID: id})
		if err == nil && n == 0 {
			return ErrNoTracker
		}
		return err
	})
	if err != nil {
		return err
	}
	x.cfg.Trackers.Publish(ctx, TrackerChanged{TrackerID: id.String()})
	return nil
}

// MergeTracker moves every key, reach and host of from to into, has each
// Repoint move its rows, and deletes from, in one Write. into keeps its
// name, and takes from's privacy while its own is unknown. It fails with
// ErrMergeSelf or ErrNoTracker.
func (x *Indexers) MergeTracker(ctx context.Context, _ identity.Admin, from, into TrackerID) error {
	if from == into {
		return ErrMergeSelf
	}
	err := x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return x.merge(ctx, tx, from, into)
	})
	if err != nil {
		return err
	}
	x.cfg.Trackers.Publish(ctx, TrackerChanged{TrackerID: from.String()})
	x.cfg.Trackers.Publish(ctx, TrackerChanged{TrackerID: into.String()})
	return nil
}

func (x *Indexers) merge(ctx context.Context, tx *storage.Tx[file], from, into TrackerID) error {
	q := db.New(tx)
	gone, err := q.GetTracker(ctx, from)
	if errors.Is(err, storage.ErrNoRows) {
		return ErrNoTracker
	}
	if err != nil {
		return err
	}
	if _, err := q.GetTracker(ctx, into); errors.Is(err, storage.ErrNoRows) {
		return ErrNoTracker
	} else if err != nil {
		return err
	}
	if err := q.SetUnknownPrivacy(ctx, db.SetUnknownPrivacyParams{Privacy: gone.Privacy, ID: into}); err != nil {
		return err
	}
	if err := q.MoveTrackerKeys(ctx, db.MoveTrackerKeysParams{IntoID: into, FromID: from}); err != nil {
		return err
	}
	if err := q.MoveReach(ctx, db.MoveReachParams{IntoID: into, FromID: from}); err != nil {
		return err
	}
	if err := q.MoveTrackerHosts(ctx, db.MoveTrackerHostsParams{IntoID: into, FromID: from}); err != nil {
		return err
	}
	for _, repoint := range x.cfg.Repoints {
		if err := repoint(ctx, tx, from, into); err != nil {
			return err
		}
	}
	return q.DeleteTracker(ctx, from)
}

// AddHosts sets Tracker t's privacy from a torrent's private flag while it
// is unknown, and adds a private torrent's announce hosts to it, inside
// tx. A public torrent adds none, since it announces to shared trackers.
func AddHosts(ctx context.Context, tx *storage.Tx[file], t TrackerID, private bool, hosts []string) error {
	q := db.New(tx)
	p := Public
	if private {
		p = Private
	}
	if err := q.SetUnknownPrivacy(ctx, db.SetUnknownPrivacyParams{Privacy: string(p), ID: t}); err != nil || !private {
		return err
	}
	for _, h := range hosts {
		if h == "" {
			continue
		}
		if err := q.AddTrackerHost(ctx, db.AddTrackerHostParams{TrackerID: t, Host: h}); err != nil {
			return err
		}
	}
	return nil
}

// reached is the Tracker a Release reaches, before it is found or made.
type reached struct {
	key, name string
	privacy   Privacy
}

// reachedBy is the Tracker r reaches through idx: its definition's for a
// catalog Indexer, named and made private or public as its summary says, its source's for a feed item that names one (a Jackett
// source's id is its definition id; a Prowlarr source's is Prowlarr's
// own), and else the one the Admin named for the feed.
func reachedBy(idx Indexer, r Release) reached {
	switch src := idx.Source.(type) {
	case Definition:
		return reached{key: "definition:" + src.ID, name: cmp.Or(src.Name, src.ID), privacy: sourcePrivacy(string(src.Privacy))}
	case Feed:
		if s := r.Source; s != nil {
			key := "prowlarr:" + idx.ID.String() + ":" + s.ID
			if s.Origin == "jackett" {
				key = "definition:" + s.ID
			}
			return reached{key: key, name: s.Name, privacy: sourcePrivacy(s.Privacy)}
		}
		return reached{key: namedKey(idx.ID), name: src.Tracker, privacy: Unknown}
	}
	panic("indexers: an Indexer with no source")
}

func namedKey(id ID) string { return "named:" + id.String() }

// sourcePrivacy reads a feed source's privacy words, which Prowlarr and
// Jackett spell "semi-private", "semiPrivate" or "semiprivate".
func sourcePrivacy(s string) Privacy {
	switch fold(s) {
	case "private":
		return Private
	case "semiprivate":
		return SemiPrivate
	case "public":
		return Public
	}
	return Unknown
}

// trackerFor is the Tracker at.key reaches in tx. A key no Tracker has
// yet goes to the Tracker whose name or definition id matches at.name or
// the key's definition id, case and punctuation ignored, else to a new
// Tracker named at.name. A known at.privacy sets the Tracker's privacy
// while it is unknown. created reports a new Tracker.
func trackerFor(ctx context.Context, tx *storage.Tx[file], at reached) (id TrackerID, created bool, err error) {
	q := db.New(tx)
	id, err = q.GetTrackerOfKey(ctx, at.key)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		if id, err = match(ctx, tx, at); err != nil {
			return TrackerID{}, false, err
		}
		if created = id.IsZero(); created {
			id = TrackerID{UUID: storage.NewUUID()}
			err := q.InsertTracker(ctx, db.InsertTrackerParams{ID: id, Name: at.name, Privacy: string(at.privacy), CreatedAt: storage.Now()})
			if err != nil {
				return TrackerID{}, false, err
			}
		}
		if err := q.InsertTrackerKey(ctx, db.InsertTrackerKeyParams{Key: at.key, TrackerID: id}); err != nil {
			return TrackerID{}, false, err
		}
	case err != nil:
		return TrackerID{}, false, err
	}
	if at.privacy != Unknown {
		err = q.SetUnknownPrivacy(ctx, db.SetUnknownPrivacyParams{Privacy: string(at.privacy), ID: id})
	}
	return id, created, err
}

// reachTracker is the Tracker at names, with Indexer idx reaching it, in tx.
// news reports a Tracker made or a reach added.
func reachTracker(ctx context.Context, tx *storage.Tx[file], idx ID, at reached) (t TrackerID, news bool, err error) {
	t, created, err := trackerFor(ctx, tx, at)
	if err != nil {
		return TrackerID{}, false, err
	}
	n, err := db.New(tx).AddReach(ctx, db.AddReachParams{IndexerID: idx, TrackerID: t})
	return t, created || n > 0, err
}

// match is the first Tracker by name that at's names match, zero for none.
// A name equal to at.name matches even when it folds to nothing.
func match(ctx context.Context, r storage.Reader[file], at reached) (TrackerID, error) {
	want := namesOf(at.name, at.key)
	all, err := trackersIn(ctx, r)
	if err != nil {
		return TrackerID{}, err
	}
	for _, t := range all {
		names := namesOf(t.Name, t.Keys...)
		if t.Name == at.name || slices.ContainsFunc(want, func(n string) bool { return slices.Contains(names, n) }) {
			return t.ID, nil
		}
	}
	return TrackerID{}, nil
}

// namesOf is name and the definition ids among keys, folded, without any
// that fold to nothing.
func namesOf(name string, keys ...string) []string {
	out := []string{fold(name)}
	for _, k := range keys {
		if id, ok := strings.CutPrefix(k, "definition:"); ok {
			out = append(out, fold(id))
		}
	}
	return slices.DeleteFunc(out, func(n string) bool { return n == "" })
}

// fold is s in lower case with only its letters and digits.
func fold(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

// trackersIn is every Tracker by name, read in r.
func trackersIn(ctx context.Context, r storage.Reader[file]) ([]Tracker, error) {
	q := db.New(r)
	rows, err := q.ListTrackers(ctx)
	if err != nil {
		return nil, err
	}
	keys, err := q.ListTrackerKeys(ctx)
	if err != nil {
		return nil, err
	}
	hosts, err := q.ListTrackerHosts(ctx)
	if err != nil {
		return nil, err
	}
	reach, err := reachOf(ctx, r)
	if err != nil {
		return nil, err
	}
	out := make([]Tracker, len(rows))
	at := map[TrackerID]int{}
	for i, row := range rows {
		out[i] = Tracker{ID: row.ID, Name: row.Name, Privacy: Privacy(row.Privacy), Indexers: reach.indexers[row.ID]}
		at[row.ID] = i
	}
	for _, k := range keys {
		out[at[k.TrackerID]].Keys = append(out[at[k.TrackerID]].Keys, k.Key)
	}
	for _, h := range hosts {
		out[at[h.TrackerID]].Hosts = append(out[at[h.TrackerID]].Hosts, h.Host)
	}
	return out, nil
}

// reach is indexers_reach both ways.
type reach struct {
	indexers map[TrackerID][]ID
	trackers map[ID][]TrackerID
}

func reachOf(ctx context.Context, r storage.Reader[file]) (reach, error) {
	rows, err := db.New(r).ListReach(ctx)
	if err != nil {
		return reach{}, err
	}
	out := reach{indexers: map[TrackerID][]ID{}, trackers: map[ID][]TrackerID{}}
	for _, row := range rows {
		out.indexers[row.TrackerID] = append(out.indexers[row.TrackerID], row.IndexerID)
		out.trackers[row.IndexerID] = append(out.trackers[row.IndexerID], row.TrackerID)
	}
	return out, nil
}
