package targets

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
	"github.com/jatassi/slipstream/internal/works"
)

// Sourcing is how a Profile's Target gets its file: downloaded, made
// from a Target above it, or downloaded and made from one after a wait
// with no download.
type Sourcing = db.Sourcing

const (
	Download                    = db.Download
	Derive             Sourcing = "derive"
	DownloadThenDerive Sourcing = "download_then_derive"
)

// The wait for a download of a DownloadThenDerive row.
const (
	DefaultGrace = 3 * 24 * time.Hour
	MinGrace     = time.Hour
	MaxGrace     = 90 * 24 * time.Hour
)

// Entry is one Target of a Profile and how it is sourced. From is the
// Target whose file a made-from row is made from, nil exactly for
// Download, and Grace is how long a DownloadThenDerive row waits for a
// download, counted from its source's import.
type Entry struct {
	Target   TargetID
	Sourcing Sourcing
	From     *TargetID
	Grace    time.Duration
}

func (e Entry) equal(o Entry) bool {
	return e.Target == o.Target && e.Sourcing == o.Sourcing && e.Grace == o.Grace && (e.From == nil) == (o.From == nil) && (e.From == nil || *e.From == *o.From)
}

func entryOf(row db.TargetsProfileTarget) Entry {
	e := Entry{Target: row.TargetID, Sourcing: row.Sourcing}
	if !row.FromTargetID.IsZero() {
		e.From = &row.FromTargetID
	}
	if row.GraceSeconds != nil {
		e.Grace = time.Duration(*row.GraceSeconds) * time.Second
	}
	return e
}

// Profile is the Targets a Work wants, in Profile order: the first a file
// fills names it. Each Content Type has one default Profile, which a Work
// added without one takes.
type Profile struct {
	ID          ProfileID
	Name        string
	ContentType works.TypeID
	Default     bool
	Entries     []Entry
	Editions    Editions
	// Languages is its Wanted Languages, in the Admin's order.
	Languages []WantedLanguage
	// Works counts the Works wanting from it. Read only.
	Works int
}

// ProfileSpec is a Profile as an Admin writes it. It may hold no Target,
// and then its Works want nothing. Languages are in the Admin's order,
// none of one language and kind twice, and only for a Content Type that
// has subtitles.
type ProfileSpec struct {
	Name        string
	ContentType works.TypeID
	Entries     []Entry
	Editions    Editions
	Languages   []WantedLanguage
}

// CreateProfile checks s and creates its Profile, which is never the
// default.
func (t *Targets) CreateProfile(ctx context.Context, _ identity.Admin, s ProfileSpec) (Profile, error) {
	if _, ok := t.cfg.Types[s.ContentType]; !ok {
		return Profile{}, ErrNoType
	}
	if err := t.checkLanguages(s.ContentType, s.Languages); err != nil {
		return Profile{}, err
	}
	p := Profile{ID: ProfileID{UUID: storage.NewUUID()}, Name: s.Name, ContentType: s.ContentType, Entries: s.Entries, Editions: s.Editions, Languages: s.Languages}
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		known, err := t.CutsIn(ctx, tx)
		if err != nil {
			return err
		}
		if err := known.check(p.Editions, p.ContentType); err != nil {
			return err
		}
		return insertProfile(ctx, db.New(tx), p)
	})
	if err != nil {
		return Profile{}, err
	}
	t.cfg.Changed.Publish(ctx, changedOf(nil))
	return p, nil
}

func insertProfile(ctx context.Context, q *db.Queries, p Profile) error {
	switch taken, err := q.ProfileNameTaken(ctx, p.Name); {
	case err != nil:
		return err
	case taken:
		return ErrProfileNameTaken
	}
	if err := checkEntries(ctx, q, p.Entries); err != nil {
		return err
	}
	err := q.InsertProfile(ctx, db.InsertProfileParams{ID: p.ID, Name: p.Name, ContentType: p.ContentType, IsDefault: p.Default, Editions: listOf(p.Editions)})
	if err != nil {
		return err
	}
	if err := insertProfileCuts(ctx, q, p.ID, p.Editions); err != nil {
		return err
	}
	if err := writeLanguages(ctx, q, p.ID, p.Languages); err != nil {
		return err
	}
	return insertEntries(ctx, q, p.ID, p.Entries)
}

func insertProfileCuts(ctx context.Context, q *db.Queries, id ProfileID, e Editions) error {
	for i, c := range e.cuts {
		if err := q.InsertProfileCut(ctx, db.InsertProfileCutParams{ProfileID: id, Position: int64(i + 1), Cut: c}); err != nil {
			return err
		}
	}
	return nil
}

// checkEntries refuses a Target named twice or that doesn't exist, and a
// made-from row whose Target has no Conversion settings, whose source
// isn't a downloaded Target above it, or whose wait is outside MinGrace
// to MaxGrace.
func checkEntries(ctx context.Context, q *db.Queries, entries []Entry) error {
	made, err := readConversions(ctx, q)
	if err != nil {
		return err
	}
	for i, e := range entries {
		if slices.ContainsFunc(entries[:i], func(o Entry) bool { return o.Target == e.Target }) {
			return ErrTargetTwice
		}
		switch has, err := q.HasTarget(ctx, e.Target); {
		case err != nil:
			return err
		case !has:
			return ErrNoTarget
		}
		if err := checkSourcing(e); err != nil {
			return err
		}
		if e.Sourcing == Download {
			continue
		}
		if made[e.Target] == nil {
			return ErrNoConversion
		}
		j := slices.IndexFunc(entries[:i], func(o Entry) bool { return o.Target == *e.From })
		if j < 0 || entries[j].Sourcing == Derive {
			return ErrSourceNotAbove
		}
	}
	return nil
}

func checkSourcing(e Entry) error {
	switch e.Sourcing {
	case Download:
		if e.From == nil && e.Grace == 0 {
			return nil
		}
	case Derive:
		if e.From != nil && e.Grace == 0 {
			return nil
		}
	case DownloadThenDerive:
		switch {
		case e.From == nil:
		case e.Grace < MinGrace || e.Grace > MaxGrace || e.Grace%time.Second != 0:
			return ErrBadGrace
		default:
			return nil
		}
	}
	return ErrBadSourcing
}

// sourceBelow reports a made-from row whose source Target sits below it.
func sourceBelow(entries []Entry) bool {
	for i, e := range entries {
		if e.From != nil && slices.ContainsFunc(entries[i+1:], func(o Entry) bool { return o.Target == *e.From }) {
			return true
		}
	}
	return false
}

func insertEntries(ctx context.Context, q *db.Queries, id ProfileID, entries []Entry) error {
	for i, e := range entries {
		row := db.InsertProfileTargetParams{ProfileID: id, Position: int64(i + 1), TargetID: e.Target, Sourcing: e.Sourcing}
		if e.From != nil {
			row.FromTargetID = *e.From
		}
		if e.Sourcing == DownloadThenDerive {
			row.GraceSeconds = new(int64(e.Grace / time.Second))
		}
		err := q.InsertProfileTarget(ctx, row)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListProfiles lists every Profile by Content Type, then name.
func (t *Targets) ListProfiles(ctx context.Context, _ identity.Admin) ([]Profile, error) {
	var out []Profile
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		out, err = listProfiles(ctx, db.New(r))
		return err
	})
	return out, err
}

// GetProfile reads Profile id, ErrNoProfile when none has it.
func (t *Targets) GetProfile(ctx context.Context, by identity.Admin, id ProfileID) (Profile, error) {
	all, err := t.ListProfiles(ctx, by)
	if err != nil {
		return Profile{}, err
	}
	i := slices.IndexFunc(all, func(p Profile) bool { return p.ID == id })
	if i < 0 {
		return Profile{}, ErrNoProfile
	}
	return all[i], nil
}

// MakeDefault makes Profile id its Content Type's default. Existing Works
// keep the Profile they have.
func (t *Targets) MakeDefault(ctx context.Context, by identity.Admin, id ProfileID) (Profile, error) {
	var changed bool
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		changed, err = MakeDefaultIn(ctx, tx, id)
		return err
	})
	if err != nil {
		return Profile{}, err
	}
	if changed {
		t.cfg.Changed.Publish(ctx, changedOf(nil))
	}
	return t.GetProfile(ctx, by, id)
}

// MakeDefaultIn makes Profile id its Content Type's default in tx, writing
// only when it isn't already. changed is whether it wrote. Its caller
// publishes targets.changed after commit when it did.
func MakeDefaultIn(ctx context.Context, tx *storage.Tx[file], id ProfileID) (changed bool, err error) {
	return makeDefault(ctx, db.New(tx), id)
}

func makeDefault(ctx context.Context, q *db.Queries, id ProfileID) (changed bool, err error) {
	p, err := q.GetProfile(ctx, id)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return false, ErrNoProfile
	case err != nil:
		return false, err
	case p.IsDefault:
		return false, nil
	}
	if err := q.ClearDefault(ctx, p.ContentType); err != nil {
		return false, err
	}
	return true, q.SetDefault(ctx, id)
}

func listProfiles(ctx context.Context, q *db.Queries) ([]Profile, error) {
	rows, err := q.ListProfiles(ctx)
	if err != nil {
		return nil, err
	}
	members, err := q.ListProfileTargets(ctx)
	if err != nil {
		return nil, err
	}
	cuts, err := q.ListProfileCuts(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := q.CountWants(ctx)
	if err != nil {
		return nil, err
	}
	langs, err := languagesOf(ctx, q)
	if err != nil {
		return nil, err
	}
	held := map[ProfileID][]Entry{}
	for _, m := range members {
		held[m.ProfileID] = append(held[m.ProfileID], entryOf(m))
	}
	named := map[ProfileID][]release.CutKey{}
	for _, c := range cuts {
		named[c.ProfileID] = append(named[c.ProfileID], c.Cut)
	}
	wanting := map[ProfileID]int{}
	for _, c := range counts {
		wanting[c.ProfileID] = int(c.Works)
	}
	out := make([]Profile, len(rows))
	for i, row := range rows {
		out[i] = Profile{
			ID: row.ID, Name: row.Name, ContentType: row.ContentType, Default: row.IsDefault, Entries: held[row.ID],
			Editions: editionsOf(row.Editions, named[row.ID]), Languages: langs[row.ID], Works: wanting[row.ID],
		}
	}
	return out, nil
}
