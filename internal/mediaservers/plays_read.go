package mediaservers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

const playsPage = 50

// newest orders after every play's start, so the first page's keyset
// passes them all.
var newest = storage.FromTime(time.UnixMicro(math.MaxInt64))

func (s service) Plays(ctx context.Context, id link.ServerID, player string, cursor link.PlayID) (Plays, error) {
	var out Plays
	err := s.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := playsPageOf(ctx, db.New(r), id, player, cursor)
		if err != nil {
			return err
		}
		if len(rows) > playsPage {
			rows = rows[:playsPage]
			out.Next = rows[playsPage-1].ID.String()
		}
		out.Plays, err = playViews(ctx, r, rows)
		return err
	})
	return out, err
}

// playsPageOf is the page of server id's plays after cursor, with one
// more play than a page holds when another page follows.
func playsPageOf(ctx context.Context, q *db.Queries, id link.ServerID, player string, cursor link.PlayID) ([]db.MediaserversPlay, error) {
	if _, err := q.GetContact(ctx, id); errors.Is(err, storage.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	page := db.ListPlaysPageParams{ServerID: id, Player: player, BeforeAt: newest, N: playsPage + 1}
	if !cursor.IsZero() {
		at, err := q.GetPlayStart(ctx, db.GetPlayStartParams{ServerID: id, ID: cursor})
		if errors.Is(err, storage.ErrNoRows) {
			return nil, ErrNoSuchPlay
		} else if err != nil {
			return nil, err
		}
		page.BeforeAt, page.BeforeID = at, cursor
	}
	return q.ListPlaysPage(ctx, page)
}

func playViews(ctx context.Context, r storage.Reader[storage.Library], rows []db.MediaserversPlay) ([]PlayView, error) {
	var ws []works.WorkID
	for _, p := range rows {
		if p.Snapshot != nil {
			ws = append(ws, p.Snapshot.Work)
		}
	}
	titles, err := works.SummariesOf(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	out := make([]PlayView, len(rows))
	for i, p := range rows {
		if out[i], err = playView(p, titles); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func playView(p db.MediaserversPlay, titles map[works.WorkID]works.Summary) (PlayView, error) {
	v := PlayView{
		ID: p.ID.String(), PlayerKey: p.PlayerKey, Player: p.Player, Product: p.Product, Platform: p.Platform,
		User: p.UserName, Path: p.ServerPath, Decision: string(p.Decision), Video: string(p.Video),
		Audio: string(p.Audio), Subtitles: string(p.Subtitles), Hardware: p.Hardware,
		Received:  receivedView(p.Received),
		StartedAt: p.StartedAt.Time(), LastSeenAt: p.LastSeenAt.Time(),
	}
	if err := json.Unmarshal([]byte(p.Reasons), &v.Reasons); err != nil {
		return PlayView{}, fmt.Errorf("read the reasons of play %s: %w", p.ID, err)
	}
	if p.Snapshot != nil {
		v.Work = &PlayWorkView{WorkID: p.Snapshot.Work.String(), Title: titles[p.Snapshot.Work].Label, Label: p.Snapshot.Label}
	}
	if !p.EndedAt.IsZero() {
		v.EndedAt = api.Some(p.EndedAt.Time())
	}
	return v, nil
}

func receivedView(r link.Received) ReceivedView {
	return ReceivedView{
		Container: unlessZero(r.Container), VideoCodec: unlessZero(r.VideoCodec), Height: unlessZero(r.Height),
		Bitrate: unlessZero(r.Bitrate), AudioCodec: unlessZero(r.AudioCodec), Channels: unlessZero(r.Channels),
	}
}

// unlessZero is v, or null when the server left it unreported.
func unlessZero[T comparable](v T) api.Nullable[T] {
	var zero T
	if v == zero {
		return api.Nullable[T]{}
	}
	return api.Some(v)
}
