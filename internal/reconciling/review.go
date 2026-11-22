package reconciling

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reconciling/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Reason is why a torrent is on the review list.
type Reason string

const (
	// NotHeld: recorded, and no Node reported it after a restore.
	NotHeld Reason = "not_held"
	// TwoHold: two Nodes hold it, and both keep it.
	TwoHold Reason = "conflict"
)

// Found is what its Node found of a torrent's files.
type Found int8

const (
	NotLooked Found = -1
	NoFiles   Found = 0
	Files     Found = 1
)

// Reviewed is one torrent on the review list.
type Reviewed struct {
	Torrent downloadclients.ID
	// Name is the torrent's name, filled by ReviewPage.
	Name string
	// Node is the Node it was recorded on.
	Node   nodes.ID
	Reason Reason
	Found  Found
	Since  storage.Time
	// Readd is the Node Re-add puts it on once Reconciling ends, nil until
	// pressed.
	Readd *nodes.ID
}

// Cursor is where a page of the review list starts after: the sort key
// of the last row a page listed. Rows added or removed meanwhile never
// move it.
type Cursor struct {
	Reason  Reason
	Found   Found
	Since   storage.Time
	Torrent downloadclients.ID
}

// Filter narrows the review list to one group. A nil field is any.
type Filter struct {
	Reason *Reason
	Found  *Found
}

// Review is a page of at most n review list rows after c, nil for the
// first, and the cursor of the next page, nil after the last.
func Review(ctx context.Context, r storage.Reader[file], f Filter, c *Cursor, n int) ([]Reviewed, *Cursor, error) {
	p := db.ListReviewPageParams{AnyReason: 1, AnyFound: 1, AfterFound: -2, Page: int64(n) + 1}
	if f.Reason != nil {
		p.AnyReason, p.Reason = 0, string(*f.Reason)
	}
	if f.Found != nil {
		p.AnyFound, p.Found = 0, int64(*f.Found)
	}
	if c != nil {
		p.AfterReason, p.AfterFound, p.AfterSince, p.AfterID = string(c.Reason), int64(c.Found), c.Since.Time().UnixMicro(), c.Torrent
	}
	rows, err := db.New(r).ListReviewPage(ctx, p)
	if err != nil {
		return nil, nil, fmt.Errorf("read the review list: %w", err)
	}
	out := make([]Reviewed, 0, min(len(rows), n))
	for i, row := range rows {
		if i == n {
			last := out[n-1]
			return out, &Cursor{Reason: last.Reason, Found: last.Found, Since: last.Since, Torrent: last.Torrent}, nil
		}
		found := NotLooked
		if row.FilesFound.Valid {
			found = Found(row.FilesFound.Int64) //nolint:gosec // files_found is 0 or 1
		}
		out = append(out, Reviewed{Torrent: row.TorrentID, Node: row.NodeID, Reason: Reason(row.Reason), Found: found, Since: row.Since, Readd: row.ReaddNode})
	}
	return out, nil, nil
}

// look asks each not held torrent's Node whether its files are still in
// its save folder, once per torrent while this process runs. A Node that
// can't answer is asked again after a restart.
func (s *Service) look(ctx context.Context) error {
	var due []downloadclients.ID
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		ids, err := db.New(r).ListToLook(ctx, int64(s.cfg.Page))
		for _, id := range ids {
			if !s.looked[id] {
				due = append(due, id)
			}
		}
		return err
	})
	if err != nil {
		return fmt.Errorf("read the review list's torrents to look for: %w", err)
	}
	for _, id := range due {
		s.looked[id] = true
		found, err := s.lookOne(ctx, id)
		if err != nil {
			s.cfg.Log.Warn("SlipStream couldn't check a Node for a torrent's files", "torrent", id, "err", err)
			continue
		}
		err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			return db.New(tx).SetLooked(ctx, db.SetLookedParams{FilesFound: int64(found), At: storage.Now(), TorrentID: id})
		})
		if err != nil {
			return fmt.Errorf("record what a Node found of a torrent's files: %w", err)
		}
	}
	if len(due) > 0 {
		s.cfg.Changed.Publish(ctx, Changed{})
	}
	return nil
}

// lookOne is what torrent id's Node found of its files: all of them, or
// not all.
func (s *Service) lookOne(ctx context.Context, id downloadclients.ID) (Found, error) {
	var l downloadclients.Look
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		l, err = downloadclients.LookOf(ctx, r, id)
		return err
	})
	if err != nil || l.Folder == "" {
		return NoFiles, err
	}
	found, err := s.cfg.Look(ctx, l.Node, l.Folder, l.Files)
	if err != nil {
		return NotLooked, err
	}
	if slices.Contains(found, false) {
		return NoFiles, nil
	}
	return Files, nil
}

// Group is one group of the review list and how many torrents it holds.
type Group struct {
	Reason Reason
	Found  Found
	Count  int
}

// Groups is the review list's groups, in its order.
func Groups(ctx context.Context, r storage.Reader[file]) ([]Group, error) {
	rows, err := db.New(r).CountReview(ctx)
	if err != nil {
		return nil, fmt.Errorf("count the review list: %w", err)
	}
	out := make([]Group, len(rows))
	for i, row := range rows {
		out[i] = Group{Reason: Reason(row.Reason), Found: Found(row.Found.Int64), Count: int(row.N)} //nolint:gosec // found is -1, 0 or 1
	}
	return out, nil
}

var ErrNotReviewed = errors.New("this torrent isn't on the review list")

// Readd puts each of ids back into the intent of its Node, or of onto
// when it is set: at once outside Reconciling, and once it ends while it
// runs. Its client checks the files already there and downloads only
// what's missing.
func (s *Service) Readd(ctx context.Context, ids []downloadclients.ID, onto *nodes.ID) error {
	var touched []role.NodeID
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		_, err := q.GetState(ctx)
		later := err == nil
		if err != nil && !errors.Is(err, storage.ErrNoRows) {
			return err
		}
		now := storage.Now()
		for _, id := range ids {
			row, err := q.GetReview(ctx, id)
			if errors.Is(err, storage.ErrNoRows) {
				return ErrNotReviewed
			}
			if err != nil {
				return err
			}
			to := row.NodeID
			if onto != nil {
				to = *onto
			}
			if err := q.MarkReadd(ctx, db.MarkReaddParams{ReaddNode: &to, At: now, TorrentID: id}); err != nil {
				return err
			}
		}
		if later {
			return nil
		}
		touched, err = s.readdIn(ctx, tx)
		return err
	})
	if err != nil {
		return err
	}
	s.touch(touched)
	s.changed(ctx)
	return nil
}

// readdIn releases every torrent Re-add was pressed for onto its Node, in
// tx, and takes it off the review list.
func (s *Service) readdIn(ctx context.Context, tx *storage.Tx[file]) ([]role.NodeID, error) {
	q := db.New(tx)
	var touched []role.NodeID
	after := downloadclients.ID{}
	for {
		rows, err := q.ListReadd(ctx, db.ListReaddParams{After: after, Page: int64(s.cfg.Page)})
		if err != nil || len(rows) == 0 {
			return touched, err
		}
		ids := make([]downloadclients.ID, len(rows))
		for i, row := range rows {
			was, err := readdOne(ctx, tx, row.TorrentID, *row.ReaddNode)
			if err != nil {
				return nil, err
			}
			ids[i] = row.TorrentID
			touched = append(touched, *row.ReaddNode, was)
		}
		if err := downloadclients.Release(ctx, tx, ids); err != nil {
			return nil, err
		}
		after = rows[len(rows)-1].TorrentID
	}
}

// readdOne moves torrent onto the Node Re-add was pressed for, when it
// was recorded on another, and takes it off the review list. It returns
// the Node it was recorded on.
func readdOne(ctx context.Context, tx *storage.Tx[file], torrent downloadclients.ID, onto nodes.ID) (nodes.ID, error) {
	q := db.New(tx)
	review, err := q.GetReview(ctx, torrent)
	if err != nil {
		return nodes.ID{}, err
	}
	if onto != review.NodeID {
		if err := downloadclients.Rerecord(ctx, tx, torrent, onto, nil); err != nil {
			return nodes.ID{}, err
		}
	}
	return review.NodeID, q.DeleteReview(ctx, torrent)
}

// Forget removes each of ids from the library and fails its Grab
// restored, so its cells are wanted again. Only a torrent no Node holds
// can be forgotten.
func (s *Service) Forget(ctx context.Context, ids []downloadclients.ID) error {
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		for _, id := range ids {
			row, err := q.GetReview(ctx, id)
			if errors.Is(err, storage.ErrNoRows) || err == nil && Reason(row.Reason) != NotHeld {
				return ErrNotReviewed
			}
			if err != nil {
				return err
			}
		}
		if err := downloadclients.Forget(ctx, tx, ids); err != nil {
			return err
		}
		for _, id := range ids {
			if err := q.DeleteReview(ctx, id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.cfg.Forgot()
	s.changed(ctx)
	return nil
}
