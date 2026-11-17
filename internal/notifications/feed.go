package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// unreadCap is the most unread rows the bell counts.
const unreadCap = 100

// FeedRow is one row the bell showed, worded now.
type FeedRow struct {
	Seq   Seq
	At    time.Time
	Kind  entry.Kind
	Title string
	// Line is the message's first line, "" for none.
	Line string
	// Link is absolute on the Address, "" while the Address isn't set.
	Link string
	Read bool
}

type Feed struct {
	// Rows are newest first.
	Rows []FeedRow
	// Unread is nil for the API key, and at most unreadCap.
	Unread *int
	// Before is the next page's cursor, 0 at the end.
	Before Seq
}

// ErrNeedsAccount is Seen called with the API key, which has no cursor.
var ErrNeedsAccount = identity.ErrNeedsAccount

// Feed renders up to limit shown rows below before, or the newest when
// before is 0. A Health row renders the entry's items cut to the ones the
// row showed, whatever the bell's switches say now.
func (s *Service) Feed(ctx context.Context, by identity.Admin, before Seq, limit int) (Feed, error) {
	if before == 0 {
		before = 1<<63 - 1
	}
	var out Feed
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		rows, err := q.ListFeed(ctx, db.ListFeedParams{Before: before, Limit: int64(limit)})
		if err != nil {
			return err
		}
		cursor, err := s.cursor(ctx, r, by)
		if err != nil {
			return err
		}
		if out.Unread, err = unread(ctx, q, by, cursor); err != nil {
			return err
		}
		events, err := shownEvents(ctx, r, rows)
		if err != nil {
			return err
		}
		l, err := linkingOf(ctx, r, events)
		if err != nil {
			return err
		}
		now := time.Now()
		out.Rows = make([]FeedRow, 0, len(rows))
		for _, row := range rows {
			fr, err := s.feedRow(events[row.EventID], row, l, now)
			if err != nil {
				return err
			}
			fr.Read = fr.Seq <= cursor
			out.Rows = append(out.Rows, fr)
		}
		if len(rows) == limit && limit > 0 {
			out.Before = rows[len(rows)-1].Seq
		}
		return nil
	})
	return out, err
}

// unread is how many rows above cursor by hasn't read, nil for the API key.
func unread(ctx context.Context, q *db.Queries, by identity.Admin, cursor Seq) (*int, error) {
	if _, ok := by.Account(); !ok {
		return nil, nil
	}
	n, err := q.CountAbove(ctx, db.CountAboveParams{Seq: cursor, Cap: unreadCap})
	count := int(n)
	return &count, err
}

func shownEvents(ctx context.Context, r storage.Reader[file], rows []db.NotificationsShown) (map[EventID]event, error) {
	heads := make([]head, len(rows))
	for i, row := range rows {
		heads[i] = head{ID: row.EventID, Kind: row.Kind, At: row.At.Time()}
	}
	return eventsOf(ctx, r, heads)
}

// linking is what a row's link reads: the Address, and which of the
// rows' Works are still in the library. A row about a deleted Work has no
// link, as on History.
type linking struct {
	address identity.Address
	kept    map[works.WorkID]bool
}

func linkingOf(ctx context.Context, r storage.Reader[file], events map[EventID]event) (linking, error) {
	address, err := identity.ReadAddress(ctx, r)
	if err != nil {
		return linking{}, err
	}
	var ids []works.WorkID
	for _, e := range events {
		if !e.Work.IsZero() {
			ids = append(ids, e.Work)
		}
	}
	kept, err := works.Existing(ctx, r, ids)
	return linking{address: address, kept: kept}, err
}

func (s *Service) feedRow(e event, row db.NotificationsShown, l linking, now time.Time) (FeedRow, error) {
	shown, err := keysFrom(row.Items)
	if err != nil {
		return FeedRow{}, err
	}
	if e, err = e.cut(shown); err != nil {
		return FeedRow{}, err
	}
	msg, err := s.render(e, now, "")
	if err != nil {
		return FeedRow{}, err
	}
	fr := FeedRow{Seq: row.Seq, At: e.At, Kind: e.Kind, Title: msg.Title}
	if len(msg.Lines) > 0 {
		fr.Line = msg.Lines[0]
	}
	if msg.Link.Path != "" && !l.address.IsZero() && (e.Work.IsZero() || l.kept[e.Work]) {
		fr.Link = l.address.String() + msg.Link.Path
	}
	return fr, nil
}

// cursor is the newest seq by has read: its stored cursor, or with none
// the newest row shown by the time its Account was created. The API key
// has read nothing.
func (s *Service) cursor(ctx context.Context, r storage.Reader[file], by identity.Admin) (Seq, error) {
	account, ok := by.Account()
	if !ok {
		return 0, nil
	}
	q := db.New(r)
	id := account.Profile().ID
	seq, err := q.GetSeen(ctx, id)
	if !errors.Is(err, storage.ErrNoRows) {
		return seq, err
	}
	created, err := identity.CreatedAt(ctx, r, id)
	if err != nil {
		return 0, err
	}
	seq, err = q.NewestSeqAt(ctx, created)
	if errors.Is(err, storage.ErrNoRows) {
		return 0, nil
	}
	return seq, err
}

// Seen moves by's cursor to through, or to the newest row when through is
// past it, never back. It tells by's other screens after the commit.
func (s *Service) Seen(ctx context.Context, by identity.Admin, through Seq) error {
	account, ok := by.Account()
	if !ok {
		return ErrNeedsAccount
	}
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		cursor, err := s.cursor(ctx, tx, by)
		if err != nil {
			return err
		}
		newest, err := q.NewestSeq(ctx)
		if errors.Is(err, storage.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		return q.UpsertSeen(ctx, db.UpsertSeenParams{AccountID: account.Profile().ID, Seq: max(cursor, min(through, newest))})
	})
	if err != nil {
		return err
	}
	s.cfg.Seen.Publish(ctx, account.Profile().ID.UUID, Seen{})
	return nil
}
