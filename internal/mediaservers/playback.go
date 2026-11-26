package mediaservers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/mediaservers/playback"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func (s service) Playback(ctx context.Context, id link.ServerID) (playback.Report, error) {
	now := time.Now()
	var in playback.Input
	var reaching role.NodeID
	err := s.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		q := db.New(r)
		srv, err := q.GetServer(ctx, id)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		set, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		w := window(now, KeepPlays.Value(set.KeepPlaysSeconds), srv.PlaysSince.Time())
		rows, err := q.ListPlaysSince(ctx, db.ListPlaysSinceParams{ServerID: id, Since: storage.FromTime(w.Since)})
		if err != nil {
			return err
		}
		plays, err := playsOf(rows)
		if err != nil {
			return err
		}
		sib, err := s.m.cfg.Siblings(ctx, r, playback.ItemsToJudge(plays))
		if err != nil {
			return err
		}
		listed, err := q.ListIndexedFiles(ctx, db.ListIndexedFilesParams{ServerID: id, Files: fileIDsJSON(sib)})
		if err != nil {
			return err
		}
		all, err := s.m.cfg.Targets(ctx, r)
		if err != nil {
			return err
		}
		titles, err := works.SummariesOf(ctx, r, playback.WorksOf(plays, sib))
		if err != nil {
			return err
		}
		in = playback.Input{Kind: srv.Kind, Window: w, Plays: plays, Siblings: sib.WithListed(itemKeysOf(listed)), Targets: all, Works: titles}
		reaching = srv.ReachingNodeID
		return nil
	})
	if err != nil {
		return playback.Report{}, err
	}
	_, in.Reachable = s.m.cfg.Facts(reaching)
	return playback.Compute(in), nil
}

func playsOf(rows []db.MediaserversPlay) ([]playback.Play, error) {
	out := make([]playback.Play, len(rows))
	for i, p := range rows {
		var reasons []string
		if err := json.Unmarshal([]byte(p.Reasons), &reasons); err != nil {
			return nil, fmt.Errorf("read a play's reasons: %w", err)
		}
		out[i] = playback.Play{
			ID: p.ID, File: p.FileID, ItemKey: p.ItemKey, PlayerKey: p.PlayerKey, Player: p.Player, Platform: p.Platform,
			Product: p.Product, User: p.UserName, Decision: p.Decision,
			Streams: link.Streams{Video: p.Video, Audio: p.Audio, Subtitles: p.Subtitles},
			Reasons: reasons, Received: p.Received, Snapshot: p.Snapshot,
			Started: p.StartedAt.Time(), LastSeen: p.LastSeenAt.Time(),
		}
	}
	return out, nil
}

func itemKeysOf(rows []db.ListIndexedFilesRow) map[files.ID]string {
	out := make(map[files.ID]string, len(rows))
	for _, l := range rows {
		out[l.FileID] = l.ItemKey
	}
	return out
}

func fileIDsJSON(s playback.Siblings) string {
	ids := []string{}
	for _, sibs := range s.Files {
		for _, sib := range sibs {
			ids = append(ids, sib.File.String())
		}
	}
	b, _ := json.Marshal(ids) //nolint:errchkjson // a string slice always marshals
	return string(b)
}
