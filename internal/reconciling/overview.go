package reconciling

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reconciling/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// Restored is the restore a Reconciling runs for.
type Restored struct {
	Cause      Cause
	Backup     string
	TakenAt    time.Time
	StartedAt  time.Time
	NewMachine bool
	// Rehome: the old Controller's folders aren't found yet.
	Rehome bool
}

// Holder is one Node that keeps its torrents until its Holdings are
// adopted, or whose adoption gave it the Downloading Role.
type Holder struct {
	Node  nodes.ID
	Cause Cause
	Since time.Time
	// Held is how many torrents its Holdings listed.
	Held     int
	Reported bool
	Adopted  bool
	// Granted: adoption gave it the Downloading Role.
	Granted bool
	// Refused is why SlipStream refused its Holdings, "" when it didn't.
	Refused   string
	Continued bool
}

// Root is one library root Reconciling waits on.
type Root struct {
	Root      roots.ID
	Saved     bool
	Scanned   bool
	Continued bool
}

// Overview is Reconciling as its page reads it: the restore, nil outside
// one, each Node it waits on or gave a Role, and each root. Outside a
// restore it lists only the Nodes that keep their torrents.
type Overview struct {
	Restore *Restored
	Nodes   []Holder
	Roots   []Root
}

func overview(ctx context.Context, r storage.Reader[file]) (Overview, error) {
	q := db.New(r)
	var o Overview
	state, err := q.GetState(ctx)
	switch {
	case errors.Is(err, storage.ErrNoRows):
	case err != nil:
		return Overview{}, fmt.Errorf("read Reconciling: %w", err)
	default:
		o.Restore = &Restored{
			Cause: Cause(state.Cause), Backup: state.BackupName, TakenAt: state.BackupTakenAt.Time(), StartedAt: state.StartedAt.Time(),
			NewMachine: state.NewMachine == 1, Rehome: state.RehomeSavedAt.IsZero(),
		}
	}
	rows, err := q.ListNodes(ctx)
	if err != nil {
		return Overview{}, fmt.Errorf("read Reconciling: %w", err)
	}
	for _, row := range rows {
		if o.Restore == nil && !row.AdoptedAt.IsZero() {
			continue
		}
		k := Holder{
			Node: row.NodeID, Cause: Cause(row.Cause), Since: row.Since.Time(), Held: int(row.Held), Reported: !row.ReportedAt.IsZero(),
			Adopted: !row.AdoptedAt.IsZero(), Granted: !row.GrantedAt.IsZero(), Continued: !row.ContinuedWithoutAt.IsZero(),
		}
		if row.Refused != nil {
			k.Refused = *row.Refused
		}
		o.Nodes = append(o.Nodes, k)
	}
	rootRows, err := q.ListRoots(ctx)
	if err != nil {
		return Overview{}, fmt.Errorf("read Reconciling: %w", err)
	}
	for _, row := range rootRows {
		o.Roots = append(o.Roots, Root{Root: row.RootID, Saved: !row.SavedAt.IsZero(), Scanned: !row.ScannedAt.IsZero(), Continued: !row.ContinuedWithoutAt.IsZero()})
	}
	return o, nil
}

// Page is the Overview with what the Reconciling page names it by.
type Page struct {
	Overview
	// Names is every Node's name, and Words each root's words.
	Names map[nodes.ID]string
	Words map[roots.ID]string
	// Standing is each Node's link, for a Node that waits.
	Standing map[nodes.ID]Standing
	Unknown  []nodes.Unknown
	Groups   []Group
}

// Standing is how a Node stands with this Controller, and since when.
type Standing struct {
	Standing nodes.Standing
	Since    time.Time
}

// Page reads the Reconciling page in one snapshot.
func (s *Service) Page(ctx context.Context) (Page, error) {
	var p Page
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if p.Overview, err = overview(ctx, r); err != nil {
			return err
		}
		named, err := nodes.Names(ctx, r)
		if err != nil {
			return err
		}
		p.Names = make(map[nodes.ID]string, len(named))
		for _, n := range named {
			p.Names[n.ID] = n.Name
		}
		if p.Words, err = s.cfg.RootWords(ctx, r); err != nil {
			return err
		}
		p.Groups, err = Groups(ctx, r)
		return err
	})
	if err != nil {
		return Page{}, err
	}
	p.Standing = make(map[nodes.ID]Standing, len(p.Nodes))
	for _, k := range p.Nodes {
		st, since := s.cfg.Roster.Standing(k.Node)
		p.Standing[k.Node] = Standing{Standing: st, Since: since}
	}
	if p.Restore != nil {
		p.Unknown = s.cfg.Roster.Unknowns()
	}
	return p, nil
}

// ReviewPage is Review in its own Read.
func (s *Service) ReviewPage(ctx context.Context, f Filter, c *Cursor, n int) (rows []Reviewed, next *Cursor, err error) {
	err = s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		if rows, next, err = Review(ctx, r, f, c, n); err != nil {
			return err
		}
		ids := make([]downloadclients.ID, len(rows))
		for i, row := range rows {
			ids[i] = row.Torrent
		}
		attached, err := downloadclients.Outcomes(ctx, r, ids)
		for i := range rows {
			rows[i].Name = attached[rows[i].Torrent].Name
		}
		return err
	})
	return rows, next, err
}
