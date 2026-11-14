package mediaserversapi_test

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	wire "github.com/jatassi/slipstream/internal/providers/jellyfin"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// family is Den with a second user, guest, beside ada.
func family() jellyfin.Server {
	srv := den(floorVer)
	srv.Users = append(srv.Users, jellyfin.User{Name: "guest", Password: "guest-password"})
	return srv
}

const position = int64(20 * time.Minute / 100)

// viewer reads and writes Den's watch data as its users do in Jellyfin.
type viewer struct {
	t          *testing.T
	c          *wire.Client
	ada, guest string
}

func (s shelf) viewer(t *testing.T) viewer {
	t.Helper()
	base, err := url.Parse(s.den.Address.URL(""))
	if err != nil {
		t.Fatal(err)
	}
	v := viewer{t: t, c: wire.New(base, wire.Jellyfin, denKey, wire.App{Client: "test", Device: "test", DeviceID: "test", Version: "1"}, s.den.Client)}
	users, err := v.c.Users(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range users {
		if u.Name == "guest" {
			v.guest = u.ID
		} else {
			v.ada = u.ID
		}
	}
	return v
}

// item is the id of Den's item at path, a path under Shelves.
func (v viewer) item(path string) string {
	v.t.Helper()
	page, err := v.c.Items(v.t.Context(), wire.Query{User: v.ada, Recursive: true})
	if err != nil {
		v.t.Fatal(err)
	}
	i := slices.IndexFunc(page.Items, func(it wire.Item) bool { return it.Path == p+path })
	if i < 0 {
		v.t.Fatalf("Den holds no item at %s", path)
	}
	return page.Items[i].ID
}

// watch marks the item at path played for ada, part way and a favorite
// for guest, and puts it in the collection Holiday.
func (v viewer) watch(path string) {
	v.t.Helper()
	id, ctx := v.item(path), v.t.Context()
	if err := v.c.MarkPlayed(ctx, v.ada, id); err != nil {
		v.t.Fatal(err)
	}
	v.guestAt(path, position)
	if _, err := v.c.CreateCollection(ctx, "Holiday", []string{id}); err != nil {
		v.t.Fatal(err)
	}
}

func (v viewer) guestAt(path string, at int64) {
	v.t.Helper()
	if err := v.c.SetUserData(v.t.Context(), v.guest, v.item(path), wire.UserData{PlaybackPositionTicks: at, IsFavorite: true}); err != nil {
		v.t.Fatal(err)
	}
}

// kept checks that the item at path holds what watch left, with guest at
// position.
func (v viewer) kept(path string) {
	v.t.Helper()
	id, ctx := v.item(path), v.t.Context()
	ada, err := v.c.UserData(ctx, v.ada, id)
	if err != nil {
		v.t.Fatal(err)
	}
	guest, err := v.c.UserData(ctx, v.guest, id)
	if err != nil {
		v.t.Fatal(err)
	}
	if !ada.Played || guest.Played || !guest.IsFavorite || guest.PlaybackPositionTicks != position {
		v.t.Errorf("the item at %s holds ada %+v and guest %+v", path, ada, guest)
	}
	sets, err := v.c.Items(ctx, wire.Query{User: v.ada, Recursive: true, Types: []string{wire.TypeBoxSet}})
	if err != nil || len(sets.Items) != 1 {
		v.t.Fatalf("Den's collections are %+v, %v", sets, err)
	}
	in, err := v.c.Items(ctx, wire.Query{User: v.ada, Parent: sets.Items[0].ID})
	if err != nil {
		v.t.Fatal(err)
	}
	if !slices.ContainsFunc(in.Items, func(it wire.Item) bool { return it.ID == id }) {
		v.t.Errorf("Holiday doesn't hold the item at %s", path)
	}
}

// move moves the bytes at from to to in media, as the disk queue does.
func move(t *testing.T, media, from, to string) {
	t.Helper()
	dst := filepath.Join(media, filepath.FromSlash(to))
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(media, filepath.FromSlash(from)), dst); err != nil {
		t.Fatal(err)
	}
}

// rename is file's pending rename "from>to", its bytes still at from.
func rename(t *testing.T, file files.ID, path string) files.Rename {
	t.Helper()
	from, to, _ := strings.Cut(path, ">")
	return files.Rename{File: file, From: mustRel(t, from), To: mustRel(t, to), Leaving: true}
}

func (s shelf) prepare(t *testing.T, rs ...files.Rename) {
	t.Helper()
	for i := range rs {
		rs[i].Root = s.root
	}
	s.w.Prepare(t, rs...)
}

// renamed moves the bytes of a prepared rename on Den and tells the
// queue, as the disk queue's Write does.
func (s shelf) renamed(t *testing.T, file files.ID, path string) {
	t.Helper()
	from, to, _ := strings.Cut(path, ">")
	move(t, s.den.Media, "Shelves/"+from, "Shelves/"+to)
	s.tell(t, at(t, files.PathMoved, file, path))
}

func expectWords(t *testing.T, c mediaservers.ChangeView, state, words string) {
	t.Helper()
	if c.State != state || c.Words != words {
		t.Errorf("the change at %s is %s: %q, want %s: %q", c.Path, c.State, c.Words, state, words)
	}
}

const (
	keptWords  = "Sent the rename. Kept every user's watch data."
	savedWords = "Saved every user's watch data before the rename. SlipStream sends this change once the rename finishes."
)

// TestARenameKeepsEveryUsersWatchData renames a file both of Den's users
// watched: before the bytes move SlipStream saves their data, and once
// Den holds the new item it holds the same data and collection.
func TestARenameKeepsEveryUsersWatchData(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelfOf(t, family(), "", p+"Trip/Trip.mkv")
		v := s.viewer(t)
		v.watch("Trip/Trip.mkv")
		trip := fileIDs{}.of("trip")
		s.prepare(t, rename(t, trip, "Trip/Trip.mkv>Trip/Trip (2024).mkv"))
		expectWords(t, change(t, s.changes(t), "Trip/Trip (2024).mkv"), "prepared", savedWords)
		s.renamed(t, trip, "Trip/Trip.mkv>Trip/Trip (2024).mkv")
		settle(5 * time.Minute)
		expectWords(t, change(t, s.changes(t), "Trip/Trip (2024).mkv"), "sent", keptWords)
		expectReported(t, s.den, "Deleted "+p+"Trip/Trip.mkv", "Created "+p+"Trip/Trip (2024).mkv")
		v.kept("Trip/Trip (2024).mkv")
	})
}

// TestARenameResumesAtItsStepThroughTheController loses seedbox's answer
// to the step after Den took the old path's removal, then takes seedbox
// offline: the Controller carries on from that step at Den's fallback,
// with no attempt counted, and the data the snapshot saved before the
// bytes moved reaches the new item.
func TestARenameResumesAtItsStepThroughTheController(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelfOf(t, family(), "", p+"Trip/Trip.mkv")
		v := s.viewer(t)
		v.watch("Trip/Trip.mkv")
		trip := fileIDs{}.of("trip")
		s.prepare(t, rename(t, trip, "Trip/Trip.mkv>Trip/Trip (2024).mkv"))
		s.w.LoseAnswer(s.w.Seedbox, func(j *msv1.Job) bool { return j.GetCarry() != nil && j.GetStep() == "old_removed" })
		s.renamed(t, trip, "Trip/Trip.mkv>Trip/Trip (2024).mkv")
		settle(75 * time.Second)
		c := change(t, s.changes(t), "Trip/Trip (2024).mkv")
		if c.State != "due" || c.Step != "old_removed" || c.Attempts != 0 {
			t.Errorf("after the lost answer the change is %s at %s after %d attempts", c.State, c.Step, c.Attempts)
		}
		s.w.Offline(s.w.Seedbox)
		settle(5 * time.Minute)
		expectWords(t, change(t, s.changes(t), "Trip/Trip (2024).mkv"), "sent", keptWords)
		if r := reported(t, s.den); len(r) < 2 || r[0] != "Deleted "+p+"Trip/Trip.mkv" || r[len(r)-1] != "Created "+p+"Trip/Trip (2024).mkv" {
			t.Errorf("Den took the reports %q", r)
		}
		v.kept("Trip/Trip (2024).mkv")
	})
}

// TestARenameIsSnapshottedOnce prepares one rename in two disk queue
// passes while guest goes on watching in between: the second pass takes
// no snapshot, and the new item holds the data saved by the first.
func TestARenameIsSnapshottedOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelfOf(t, family(), "", p+"Trip/Trip.mkv")
		v := s.viewer(t)
		v.watch("Trip/Trip.mkv")
		trip := fileIDs{}.of("trip")
		r := rename(t, trip, "Trip/Trip.mkv>Trip/Trip (2024).mkv")
		s.prepare(t, r)
		v.guestAt("Trip/Trip.mkv", 2*position)
		s.prepare(t, r)
		if got := s.queued(t); !slices.Equal(got, []string{"moved Trip/Trip.mkv>Trip/Trip (2024).mkv"}) {
			t.Errorf("Den's queue holds %v, want the one prepared rename", got)
		}
		s.renamed(t, trip, "Trip/Trip.mkv>Trip/Trip (2024).mkv")
		settle(5 * time.Minute)
		v.kept("Trip/Trip (2024).mkv")
	})
}

// TestARenameAgainBeforeSendingKeepsTheFirstSnapshot renames a watched
// file twice inside Wait before sending: the second rename takes no
// snapshot of a path Den never held, and the one change from the first
// path to the last carries the data.
func TestARenameAgainBeforeSendingKeepsTheFirstSnapshot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelfOf(t, family(), "", p+"Trip/Trip.mkv")
		v := s.viewer(t)
		v.watch("Trip/Trip.mkv")
		trip := fileIDs{}.of("trip")
		s.prepare(t, rename(t, trip, "Trip/Trip.mkv>Trip/Trip (2024).mkv"))
		s.renamed(t, trip, "Trip/Trip.mkv>Trip/Trip (2024).mkv")
		s.prepare(t, rename(t, trip, "Trip/Trip (2024).mkv>Trip 2024/Trip (2024).mkv"))
		s.renamed(t, trip, "Trip/Trip (2024).mkv>Trip 2024/Trip (2024).mkv")
		if got := s.queued(t); !slices.Equal(got, []string{"moved Trip/Trip.mkv>Trip 2024/Trip (2024).mkv"}) {
			t.Errorf("Den's queue holds %v, want one rename from the first path to the last", got)
		}
		settle(5 * time.Minute)
		expectWords(t, change(t, s.changes(t), "Trip 2024/Trip (2024).mkv"), "sent", keptWords)
		v.kept("Trip 2024/Trip (2024).mkv")
	})
}

// TestARenameSnapshotsAtSendWhenDenWasDown prepares a rename while Den is
// down: the Admin reads that its watch data couldn't be saved, and once
// Den is back the sender saves it before Den hears of the rename. When
// Den's own scan took the old item meanwhile, the change says what it
// lost.
func TestARenameSnapshotsAtSendWhenDenWasDown(t *testing.T) {
	for _, c := range []struct {
		name  string
		scan  bool
		words string
	}{
		{"the old item is still there", false, keptWords},
		{"Den's scan took the old item", true, "Sent the rename. Lost: watch state, collections."},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s := newShelfOf(t, family(), "", p+"Trip/Trip.mkv")
				v := s.viewer(t)
				v.watch("Trip/Trip.mkv")
				trip := fileIDs{}.of("trip")
				up := s.w.Down(t, host)
				s.prepare(t, rename(t, trip, "Trip/Trip.mkv>Trip/Trip (2024).mkv"))
				ch := change(t, s.changes(t), "Trip/Trip (2024).mkv")
				expectWords(t, ch, "prepared", "SlipStream couldn't save the watch data before the rename. It tries again when it sends this change.")
				if !slices.Equal(ch.Lost, []string{"watch_state", "collections"}) {
					t.Errorf("the prepared change lists %v as lost", ch.Lost)
				}
				up()
				move(t, s.den.Media, "Shelves/Trip/Trip.mkv", "Shelves/Trip/Trip (2024).mkv")
				if c.scan {
					s.den.Cue(t, "jellyfin-scan", struct{}{})
					settle(10 * time.Second)
				}
				s.tell(t, at(t, files.PathMoved, trip, "Trip/Trip.mkv>Trip/Trip (2024).mkv"))
				settle(5 * time.Minute)
				expectWords(t, change(t, s.changes(t), "Trip/Trip (2024).mkv"), "sent", c.words)
				if !c.scan {
					v.kept("Trip/Trip (2024).mkv")
				}
			})
		})
	}
}

// TestFortyRenamesWaitOnePairOfDelays renames forty files on Den in one
// plan: Den takes one removal report and one creation report, and every
// rename is sent within Wait before sending and two monitor delays.
func TestFortyRenamesWaitOnePairOfDelays(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var placed, olds, news []string
		var rs []files.Rename
		ids := fileIDs{}
		for n := 1; n <= 40; n++ {
			old, renamed := fmt.Sprintf("Clip %02d/Clip %02d.mkv", n, n), fmt.Sprintf("Clip %02d (2024)/Clip %02d (2024).mkv", n, n)
			placed, olds, news = append(placed, p+old), append(olds, "Deleted "+p+old), append(news, "Created "+p+renamed)
			rs = append(rs, rename(t, ids.of(old), old+">"+renamed))
		}
		s := newShelfOf(t, family(), "", placed...)
		s.prepare(t, rs...)
		start := time.Now()
		for _, r := range rs {
			s.renamed(t, r.File, r.From.String()+">"+r.To.String())
		}
		settle(10*time.Second + 2*time.Minute + 30*time.Second)
		sent := 0
		for _, c := range s.changes(t) {
			if c.Kind == "moved" && c.State == "sent" && c.Words == keptWords {
				sent++
			}
		}
		if sent != 40 {
			t.Errorf("%s after the renames %d of 40 read %q", time.Since(start), sent, keptWords)
		}
		expectReported(t, s.den, strings.Join(olds, ", "), strings.Join(news, ", "))
	})
}

// TestAPutBackOrCalledOffRenameDropsItsPreparedChange prepares two
// renames: one is put back, and the next pass lists neither, so Den's
// queue drops both and Den hears of nothing.
func TestAPutBackOrCalledOffRenameDropsItsPreparedChange(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelfOf(t, family(), "", p+"Trip/Trip.mkv", p+"Hike/Hike.mkv")
		ids := fileIDs{}
		s.prepare(t, rename(t, ids.of("trip"), "Trip/Trip.mkv>Trip/Trip (2024).mkv"), rename(t, ids.of("hike"), "Hike/Hike.mkv>Hike/Hike (2023).mkv"))
		if got := s.queued(t); len(got) != 2 {
			t.Fatalf("Den's queue holds %v, want both prepared renames", got)
		}
		s.tell(t, at(t, files.PathUnmoved, ids.of("trip"), "Trip/Trip.mkv"))
		if got := s.queued(t); !slices.Equal(got, []string{"moved Hike/Hike.mkv>Hike/Hike (2023).mkv"}) {
			t.Errorf("after the put back Den's queue holds %v", got)
		}
		s.prepare(t)
		if got := s.queued(t); len(got) != 0 {
			t.Errorf("after a pass with no rename Den's queue holds %v", got)
		}
		settle(10 * time.Minute)
		expectReported(t, s.den)
	})
}

// TestAnUpgradeUnderANewNameKeepsEveryUsersWatchData lands an upgrade's
// new file under another name: its old file's item is snapshotted before
// the old bytes go, and the one replaced change carries the data to the
// new file's item.
func TestAnUpgradeUnderANewNameKeepsEveryUsersWatchData(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelfOf(t, family(), "", p+"Trip/Trip - 1080p.mkv")
		v := s.viewer(t)
		v.watch("Trip/Trip - 1080p.mkv")
		ids := fileIDs{}
		place(t, s.den.Media, "Shelves/Trip/Trip - 2160p.mkv")
		added := at(t, files.PathAdded, ids.of("new"), "Trip/Trip - 2160p.mkv")
		added.Upgrade = true
		s.tell(t, added)
		up := rename(t, ids.of("old"), "Trip/Trip - 1080p.mkv>Trip/Trip - 2160p.mkv")
		up.ReplacedBy = ids.of("new")
		s.prepare(t, up)
		if err := os.Remove(filepath.Join(s.den.Media, "Shelves", "Trip", "Trip - 1080p.mkv")); err != nil {
			t.Fatal(err)
		}
		removed := at(t, files.PathRemoved, ids.of("old"), "Trip/Trip - 1080p.mkv")
		removed.ReplacedBy = ids.of("new")
		s.tell(t, removed)
		if got := s.queued(t); !slices.Equal(got, []string{"replaced Trip/Trip - 1080p.mkv>Trip/Trip - 2160p.mkv"}) {
			t.Errorf("Den's queue holds %v, want the one replaced carry", got)
		}
		settle(5 * time.Minute)
		expectWords(t, change(t, s.changes(t), "Trip/Trip - 2160p.mkv"), "sent", "Sent the replaced file. Kept every user's watch data.")
		v.kept("Trip/Trip - 2160p.mkv")
	})
}

// plexKey is the ratingKey of Living room Plex's item at rel, a path
// under Shows.
func plexKey(t *testing.T, px mstest.Plex, rel string) string {
	t.Helper()
	base, err := url.Parse(homeConn)
	if err != nil {
		t.Fatal(err)
	}
	c := plex.New(base, plexToken, plex.Headers{}, px.Server.From(t, "home"))
	at := `C:\Media\Shows\` + strings.ReplaceAll(rel, "/", `\`)
	for _, typ := range plex.ItemTypes {
		page, err := c.SectionPage(t.Context(), "2", typ, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range page.Items {
			for _, v := range m.Media {
				if slices.ContainsFunc(v.Parts, func(p plex.Part) bool { return p.File == at }) {
					return string(m.RatingKey)
				}
			}
		}
	}
	t.Fatalf("Plex holds no item at %s", rel)
	return ""
}

// plexWatch scrobbles the item at rel and tags it with the collection
// Pilots, as a person does in Plex.
func plexWatch(t *testing.T, px mstest.Plex, rel string) {
	t.Helper()
	key := plexKey(t, px, rel)
	plexSend(t, px, http.MethodGet, "/:/scrobble", url.Values{"key": {key}, "identifier": {"com.plexapp.plugins.library"}})
	plexSend(t, px, http.MethodPut, "/library/metadata/"+key, url.Values{"collection[0].tag.tag": {"Pilots"}})
}

func plexSend(t *testing.T, px mstest.Plex, method, path string, q url.Values) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, homeConn+path+"?"+q.Encode(), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Plex-Token", plexToken)
	resp, err := px.Server.From(t, "home").Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s %s = %d", method, path, resp.StatusCode)
	}
}

func (s shelf) plexRenamed(t *testing.T, px mstest.Plex, file files.ID, path string) {
	t.Helper()
	from, to, _ := strings.Cut(path, ">")
	move(t, px.Server.Media, "Shows/"+from, "Shows/"+to)
	s.tell(t, at(t, files.PathMoved, file, path))
}

// TestARenameInItsFolderKeepsThePlexItem renames a watched file in its
// folder: Plex refreshes that folder once, keeps the item, and the Admin
// reads that every user's watch data was kept.
func TestARenameInItsFolderKeepsThePlexItem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const old, renamed = "Voyage/Season 01/Voyage - S01E01.mkv", "Voyage/Season 01/Voyage - S01E01 - Pilot.mkv"
		s, px := newShows(t, old)
		plexWatch(t, px, old)
		key := plexKey(t, px, old)
		f := fileIDs{}.of("e1")
		s.prepare(t, rename(t, f, old+">"+renamed))
		s.plexRenamed(t, px, f, old+">"+renamed)
		settle(time.Minute)
		expectWords(t, change(t, s.changes(t), renamed), "sent", keptWords)
		want := []string{"section 2", `section 2 at C:\Media\Shows\Voyage\Season 01`}
		if got := refreshed(t, px); !slices.Equal(got, want) {
			t.Errorf("Plex took the refreshes %v, want %v", got, want)
		}
		if got := plexKey(t, px, renamed); got != key {
			t.Errorf("the renamed file is item %s, want the item it was, %s", got, key)
		}
	})
}

// TestAMoveAcrossFoldersListsOnlyWhatDidntCarry moves two watched files
// to another folder, one under a new number, while a third file leaves
// the old folder: Plex refreshes the new folder first, and the old one
// only after the moves are confirmed. The move Plex carried keeps its
// data, and the other lists what it lost.
func TestAMoveAcrossFoldersListsOnlyWhatDidntCarry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const e1, e2, e3 = "Voyage/Season 01/Voyage - S01E01.mkv", "Voyage/Season 01/Voyage - S01E02.mkv", "Voyage/Season 01/Voyage - S01E03.mkv"
		const to1, to2 = "Voyage/Season 1/Voyage - S01E01.mkv", "Voyage/Season 1/Voyage - S01E09.mkv"
		s, px := newShows(t, e1, e2, e3)
		plexWatch(t, px, e1)
		plexWatch(t, px, e2)
		ids := fileIDs{}
		s.prepare(t, rename(t, ids.of("e1"), e1+">"+to1), rename(t, ids.of("e2"), e2+">"+to2))
		if err := os.Remove(filepath.Join(px.Server.Media, "Shows", filepath.FromSlash(e3))); err != nil {
			t.Fatal(err)
		}
		s.tell(t, at(t, files.PathRemoved, ids.of("e3"), e3))
		s.plexRenamed(t, px, ids.of("e1"), e1+">"+to1)
		s.plexRenamed(t, px, ids.of("e2"), e2+">"+to2)
		settle(5 * time.Minute)
		cs := s.changes(t)
		expectWords(t, change(t, cs, to1), "sent", keptWords)
		lost := change(t, cs, to2)
		expectWords(t, lost, "sent", "Sent the rename. Lost: watch state, collections.")
		if !slices.Equal(lost.Lost, []string{"watch_state", "collections"}) {
			t.Errorf("the second move lists %v as lost", lost.Lost)
		}
		want := []string{"section 2", `section 2 at C:\Media\Shows\Voyage\Season 1`, `section 2 at C:\Media\Shows\Voyage\Season 01`, `section 2 at C:\Media\Shows\Voyage\Season 01`}
		if got := refreshed(t, px); !slices.Equal(got, want) {
			t.Errorf("Plex took the refreshes %v, want %v", got, want)
		}
	})
}
