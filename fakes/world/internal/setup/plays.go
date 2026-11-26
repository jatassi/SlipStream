package setup

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/fakes/internal/serverfake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/node"
)

// Plays plays each Scenario server's plays through its fake. It waits for
// every Work a play names to be imported and for the server to list it,
// starts every play at once, waits until each is seen on a later look,
// with a snapshot when its file is in a root, then stops them all and
// waits until each has ended.
func Plays(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready, claims Claims, mediaDir string, w MediaWorld) error {
	controller := controllerURL(s, ready)
	var playing []serverfake.Played
	for _, kind := range scenario.MediaServerKinds {
		p, ok, err := serverfake.Playing(s, kind, versions[kind])
		if err != nil {
			return err
		}
		if ok && len(p.Plays) > 0 {
			playing = append(playing, p)
		}
	}
	if len(playing) == 0 {
		return nil
	}
	session, err := nodeapi.SignIn(ctx, c, controller, s.Admin.Email, s.Admin.PIN)
	if err != nil {
		return err
	}
	defer func() { _ = nodeapi.SignOut(context.WithoutCancel(ctx), c, controller, session) }()
	ids, err := nodeapi.MediaServerIDs(ctx, c, controller, claims.APIKey)
	if err != nil {
		return err
	}
	works, err := playedWorks(ctx, c, s, controller, claims.APIKey, session, mediaDir)
	if err != nil {
		return err
	}
	for _, p := range playing {
		if err := play(ctx, c, s, controller, claims.APIKey, session, ids[p.Name], p, works, w); err != nil {
			return fmt.Errorf("media server %s: %w", p.Name, err)
		}
	}
	return nil
}

// AwaitPlayedFiles waits until every rooted file the Scenario's plays name
// is a Work's file, so the servers added next index each with its file.
func AwaitPlayedFiles(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready, claims Claims, mediaDir string) error {
	if len(playedFiles(s)) == 0 {
		return nil
	}
	controller := controllerURL(s, ready)
	session, err := nodeapi.SignIn(ctx, c, controller, s.Admin.Email, s.Admin.PIN)
	if err != nil {
		return err
	}
	defer func() { _ = nodeapi.SignOut(context.WithoutCancel(ctx), c, controller, session) }()
	_, err = playedWorks(ctx, c, s, controller, claims.APIKey, session, mediaDir)
	return err
}

// rooted is whether file, a folder/path, is in a library root.
func rooted(s scenario.Scenario, file string) bool {
	folder, _, _ := strings.Cut(file, "/")
	return slices.ContainsFunc(s.Library.Roots, func(r scenario.LibraryRoot) bool { return r.Folder == folder })
}

// playedWorks waits until every rooted file a play names is a Work's file,
// and returns each such file's Work id.
func playedWorks(ctx context.Context, c *http.Client, s scenario.Scenario, controller, apiKey string, session *http.Cookie, mediaDir string) (map[string]string, error) {
	files := playedFiles(s)
	roots, err := nodeapi.ListRoots(ctx, c, controller, apiKey)
	if err != nil {
		return nil, err
	}
	folders := map[string]string{}
	for _, r := range roots {
		if rel, err := filepath.Rel(mediaDir, r.Path); err == nil {
			folders[r.ID] = filepath.ToSlash(rel)
		}
	}
	out := map[string]string{}
	err = nodeapi.AwaitEvent(ctx, c, controller, session, "files.changed", func() (bool, error) {
		all, err := nodeapi.WorkFiles(ctx, c, controller, apiKey)
		if err != nil {
			return false, err
		}
		for work, fs := range all {
			for _, f := range fs {
				out[folders[f.RootID]+"/"+f.Path] = work
			}
		}
		return !slices.ContainsFunc(files, func(f string) bool { _, ok := out[f]; return !ok }), nil
	})
	if err != nil {
		return nil, fmt.Errorf("wait for the played files' Works: %w", err)
	}
	return out, nil
}

// playedFiles are the rooted files and versions the Scenario's plays name.
func playedFiles(s scenario.Scenario) []string {
	var files []string
	for _, kind := range scenario.MediaServerKinds {
		m, _ := s.MediaServer(kind)
		for _, p := range m.Plays {
			for _, f := range []string{p.File, p.Version} {
				if f != "" && rooted(s, f) && !slices.Contains(files, f) {
					files = append(files, f)
				}
			}
		}
	}
	return files
}

func play(ctx context.Context, c *http.Client, s scenario.Scenario, controller, apiKey string, session *http.Cookie, id string, p serverfake.Played, works map[string]string, w MediaWorld) error {
	if err := awaitListed(ctx, c, controller, apiKey, session, id, p, works); err != nil {
		return err
	}
	sessions, snapped, err := start(ctx, s, p, w)
	if err != nil {
		return err
	}
	if err := awaitPlays(ctx, c, controller, apiKey, id, func(plays []nodeapi.Play) bool {
		seen, with := 0, 0
		for _, pl := range plays {
			if pl.LastSeenAt.After(pl.StartedAt) {
				seen++
			}
			if pl.Work != nil {
				with++
			}
		}
		return len(plays) == len(sessions) && seen == len(sessions) && with == snapped
	}); err != nil {
		return fmt.Errorf("wait for %d plays to be seen: %w", len(sessions), err)
	}
	for _, key := range sessions {
		if err := w.Cue(ctx, p.Kind+"-stop", map[string]string{"session": key}, nil); err != nil {
			return err
		}
	}
	if err := awaitPlays(ctx, c, controller, apiKey, id, func(plays []nodeapi.Play) bool {
		return !slices.ContainsFunc(plays, func(pl nodeapi.Play) bool { return pl.EndedAt == nil })
	}); err != nil {
		return fmt.Errorf("wait for %d plays to end: %w", len(sessions), err)
	}
	return nil
}

// awaitListed waits until the server id lists each Work p plays, reading
// the Work page, which asks the server to look up its files.
func awaitListed(ctx context.Context, c *http.Client, controller, apiKey string, session *http.Cookie, id string, p serverfake.Played, works map[string]string) error {
	var listed []string
	for _, pl := range p.Plays {
		if work, ok := works[pl.File]; ok && !slices.Contains(listed, work) {
			listed = append(listed, work)
		}
	}
	for _, work := range listed {
		if err := nodeapi.AwaitEvent(ctx, c, controller, session, "media-servers.items-found", func() (bool, error) {
			servers, err := nodeapi.OpenIn(ctx, c, controller, apiKey, work)
			return slices.Contains(servers, id), err
		}); err != nil {
			return fmt.Errorf("wait for Work %s to be listed: %w", work, err)
		}
	}
	return nil
}

// start starts every play of p at once, each its own session, and
// returns the sessions and how many play a rooted file.
func start(ctx context.Context, s scenario.Scenario, p serverfake.Played, w MediaWorld) (sessions []string, snapped int, err error) {
	for _, pl := range p.Plays {
		args, cerr := cueOf(p, pl)
		if cerr != nil {
			return nil, 0, cerr
		}
		for range pl.Times() {
			var started struct {
				Session string `json:"session"`
			}
			if err := w.Cue(ctx, p.Kind+"-play", args, &started); err != nil {
				return nil, 0, err
			}
			sessions = append(sessions, started.Session)
			if rooted(s, pl.File) {
				snapped++
			}
		}
	}
	return sessions, snapped, nil
}

// awaitPlays polls the plays of the server id until done holds. A later
// look moves a play's last seen without any event, so it polls.
func awaitPlays(ctx context.Context, c *http.Client, controller, apiKey, id string, done func([]nodeapi.Play) bool) error {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		plays, err := nodeapi.Plays(ctx, c, controller, apiKey, id)
		if err != nil || done(plays) {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for the plays of media server %s: %w", id, context.Cause(ctx))
		case <-tick.C:
		}
	}
}

// cueOf is pl as p's <kind>-play cue takes it, reported by its player.
func cueOf(p serverfake.Played, pl scenario.ServerPlay) (serverfake.Play, error) {
	file, err := p.Path(pl.File)
	if err != nil {
		return serverfake.Play{}, err
	}
	a := serverfake.Play{
		Path: file.String(), User: pl.User, Player: pl.Player.Name, PlayerKey: pl.Player.Key, Platform: pl.Player.Platform, Product: pl.Player.Product,
		Video: link.StreamDecision(pl.Video), Audio: link.StreamDecision(pl.Audio), Subtitles: link.SubtitleDecision(pl.Subtitles),
		ContainerChanged: pl.ContainerChanged, AudioTrack: pl.AudioTrack, SubtitleTrack: pl.SubtitleTrack,
		Reasons: pl.Reasons, Hardware: pl.Hardware, Unreported: new(false),
	}
	if pl.Version != "" {
		version, err := p.Path(pl.Version)
		if err != nil {
			return serverfake.Play{}, err
		}
		a.Version = version.String()
	}
	if r := pl.Received; r != nil {
		a.Received = mediaserver.Received{Container: r.Container, VideoCodec: r.VideoCodec, Height: r.Height, Bitrate: r.Bitrate, AudioCodec: r.AudioCodec, Channels: r.Channels}
	}
	return a, nil
}
