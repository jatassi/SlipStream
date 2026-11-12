package sim_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/media/mediatest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers"
)

// copperFolder is the series folder a folder's files lie in, whose name
// matches The Copper Coast in the library.
const copperFolder = "The Copper Coast (2022)"

func episodeName(n int) string {
	return "The.Copper.Coast.S01E" + strconv.Itoa(100 + n)[1:] + ".720p.WEB-DL.H264-FKW.mkv"
}

type importView struct {
	ID    string `json:"id"`
	Files []struct {
		Position int    `json:"position"`
		Path     string `json:"path"`
		Outcome  string `json:"outcome"`
		Reason   *struct {
			Code string `json:"code"`
		} `json:"reason"`
	} `json:"files"`
}

// outcomes is each file of the Import as "name: outcome reason", by
// position.
func (v importView) outcomes() []string {
	out := make([]string, len(v.Files))
	for i, f := range v.Files {
		out[i] = filepath.Base(f.Path) + ": " + f.Outcome
		if f.Reason != nil {
			out[i] += " " + f.Reason.Code
		}
	}
	return out
}

func (v importView) count(outcome string) int {
	n := 0
	for _, f := range v.Files {
		if f.Outcome == outcome {
			n++
		}
	}
	return n
}

// crossNode is the packs world, where home holds the TV root and seedbox
// a download folder, with The Copper Coast in the library.
type crossNode struct {
	w    *sim.World
	work string
	// downloads is seedbox's download folder.
	downloads string
}

// launchCrossNode launches the world with edit on seedbox, which probes
// each episode in copperFolder as the season pack's copy of it.
func launchCrossNode(t *testing.T, edit func(e *node.Env), opts ...sim.Option) crossNode {
	t.Helper()
	s, err := scenario.Load("packs")
	if err != nil {
		t.Fatal(err)
	}
	w := sim.Launch(t, s, append([]sim.Option{sim.WithWalk(), simtest.OnSeedbox(func(e *node.Env) {
		probes, ok := e.Media.(mediatest.Prober)
		if !ok {
			t.Fatalf("seedbox probes with %T, want the walk's mediatest.Prober", e.Media)
		}
		for n := 1; n <= 10; n++ {
			probes.Of[copperFolder+"/"+episodeName(n)] = probes.Of["The.Copper.Coast.S01.720p.WEB-DL.H264-FKW/"+episodeName(n)]
		}
		edit(e)
	})}, opts...)...)
	added := simtest.Decode[struct {
		Work struct {
			ID string `json:"id"`
		} `json:"work"`
	}](t, "addWork", w.API().Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"series","key":"1101","preset":"all"}`))
	if r := w.API().Admin("home", http.MethodPatch, "/api/v1/imports/importing-settings", `{"values":{"library.importing.smallestVideo":0}}`); r.Status != http.StatusOK {
		t.Fatalf("updateImportingSettings = %d %s, want 200", r.Status, r.Body)
	}
	return crossNode{w: w, work: added.Work.ID, downloads: w.Node("seedbox").DataDir().Resolve("downloads")}
}

// write puts a file of size bytes at rel in seedbox's download folder.
func (c crossNode) write(t *testing.T, rel string, size int) {
	t.Helper()
	at := filepath.Join(c.downloads, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(at), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte(strings.Repeat(rel, size/len(rel)+1)[:size]), 0o600); err != nil {
		t.Fatal(err)
	}
}

// open opens folder in seedbox's download folder by hand and answers its
// one Import.
func (c crossNode) open(t *testing.T, folder string) string {
	t.Helper()
	body := `{"nodeId":"` + c.w.API().Node("seedbox").NodeID + `","folder":` + strconv.Quote(filepath.Join(c.downloads, folder)) + `}`
	r := c.w.API().Admin("home", http.MethodPost, "/api/v1/imports", body)
	var opened struct {
		Imports []importView `json:"imports"`
	}
	if err := json.Unmarshal([]byte(r.Body), &opened); err != nil || r.Status != http.StatusCreated {
		t.Fatalf("importFolder %s = %d %s, want 201", folder, r.Status, r.Body)
	}
	if len(opened.Imports) != 1 {
		t.Fatalf("importFolder %s opened %d Imports, want The Copper Coast's alone", folder, len(opened.Imports))
	}
	return opened.Imports[0].ID
}

func (c crossNode) get(t *testing.T, id string) importView {
	t.Helper()
	return simtest.Decode[importView](t, "getImport", c.w.API().Admin("home", http.MethodGet, "/api/v1/imports/"+id, ""))
}

// await waits in fake time until id's files read as done says.
func (c crossNode) await(t *testing.T, id, what string, done func(importView) bool) importView {
	t.Helper()
	var v importView
	for range simtest.Settled / time.Second {
		if v = c.get(t, id); done(v) {
			return v
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("%s: the Import's files still read %q", what, v.outcomes())
	return v
}

func (c crossNode) confirm(t *testing.T, id string) {
	t.Helper()
	if r := c.w.API().Admin("home", http.MethodPost, "/api/v1/imports/confirm", `{"importIds":["`+id+`"]}`); r.Status != http.StatusOK {
		t.Fatalf("confirmImports = %d %s, want 200", r.Status, r.Body)
	}
}

// importHome sends episode 4 into home's root first, so the pack's copy
// of it is no upgrade.
func (c crossNode) importHome(t *testing.T) {
	t.Helper()
	c.write(t, "early/"+copperFolder+"/"+episodeName(4), 1<<20)
	id := c.open(t, "early")
	c.await(t, id, "episode 4 waits confirm", func(v importView) bool { return v.count("waiting") == 1 })
	c.confirm(t, id)
	c.await(t, id, "episode 4 is placed", func(v importView) bool { return v.count("placed") == 1 })
}

// library is the videos home's TV root holds, by name.
func (c crossNode) library(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(filepath.Join(c.w.Manifest().MediaDir, "tv"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(p) == ".mkv" && !strings.Contains(p, ".slipstream") {
			out = append(out, d.Name())
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

// staged is every file under the TV root's staging folder.
func (c crossNode) staged(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(filepath.Join(c.w.Manifest().MediaDir, "tv", storing.StagingDir), func(_ string, d os.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return filepath.SkipAll
		}
		if err == nil && !d.IsDir() {
			out = append(out, d.Name())
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// lands holds each file of one Import before its landing on home until
// its position is let go.
type lands struct {
	mu     sync.Mutex
	gate   map[int]chan struct{}
	armed  string
	opened bool
}

func newLands() *lands { return &lands{gate: map[int]chan struct{}{}} }

func (l *lands) at(position int) chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch, ok := l.gate[position]
	if !ok {
		ch = make(chan struct{})
		l.gate[position] = ch
	}
	return ch
}

// arm holds the landings of Import id from now on.
func (l *lands) arm(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.armed = id
}

// open lets every landing go.
func (l *lands) open() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.opened {
		return
	}
	l.opened = true
	for _, ch := range l.gate {
		select {
		case <-ch:
		default:
			close(ch)
		}
	}
}

func (l *lands) let(position int) {
	ch := l.at(position)
	l.mu.Lock()
	defer l.mu.Unlock()
	close(ch)
}

func (l *lands) before(ctx context.Context, s imports.Step) {
	l.mu.Lock()
	held := s.At == imports.AtLand && s.Import.String() == l.armed && !l.opened
	l.mu.Unlock()
	if !held {
		return
	}
	select {
	case <-l.at(s.Position):
	case <-ctx.Done():
	}
}

// landInTurn lets each file of Import id waiting to land go in order, and
// checks it lands while the rest still wait.
func (c crossNode) landInTurn(t *testing.T, id string, l *lands, order []int) importView {
	t.Helper()
	v := c.await(t, id, "the other eight arrive", func(v importView) bool { return v.count("placing") == 8 })
	left := 8
	for _, position := range order {
		if v.Files[position].Outcome == "placed" {
			continue
		}
		l.let(position)
		left--
		v = c.await(t, id, "episode "+strconv.Itoa(position+1)+" lands", func(v importView) bool { return v.Files[position].Outcome == "placed" })
		if v.count("placing") != left {
			t.Fatalf("once episode %d landed the files read %q, want %d still waiting to land", position+1, v.outcomes(), left)
		}
	}
	return v
}

// TestPackLandsInAnyOrder opens a season pack of ten episodes on seedbox
// by hand, with episode 4 already in home's TV root. Only the nine files
// with cells are sent. The first to arrive lands while the rest still
// send, and once all have arrived each lands on its own, in any order,
// while the rest wait to land.
func TestPackLandsInAnyOrder(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		// Episode 4's one chunk and the pack's first go free.
		p, l := parkedAt(3), newLands()
		c := launchCrossNode(t, func(e *node.Env) { e.Test.Transfers.Sent = p.sent }, sim.Edit(func(name string, e *node.Env) {
			if name == "home" {
				e.Test.BeforeImport = l.before
			}
		}))
		t.Cleanup(func() {
			l.open()
			select {
			case <-p.release:
			default:
				close(p.release)
			}
		})
		c.importHome(t)
		for n := 1; n <= 10; n++ {
			c.write(t, "pack/"+copperFolder+"/"+episodeName(n), 1<<20)
		}
		id := c.open(t, "pack")
		v := c.await(t, id, "the pack is read", func(v importView) bool { return v.count("waiting") == 10 })
		if got := v.outcomes()[3]; got != episodeName(4)+": waiting not_an_upgrade" {
			t.Fatalf("episode 4 reads %q, want it waiting not_an_upgrade", got)
		}
		c.confirm(t, id)
		<-p.reached
		v = c.await(t, id, "the first file to arrive lands", func(v importView) bool { return v.count("placed") == 1 })
		if v.count("sending") != 8 {
			t.Fatalf("once the first file landed the files read %q, want the other eight still sending", v.outcomes())
		}
		l.arm(id)
		close(p.release)
		v = c.landInTurn(t, id, l, []int{6, 0, 9, 2, 4, 8, 1, 7, 5})
		if got := v.outcomes()[3]; got != episodeName(4)+": waiting not_an_upgrade" {
			t.Fatalf("episode 4 reads %q once the rest landed, want it still waiting not_an_upgrade and never sent", got)
		}
		if got := c.library(t); len(got) != 10 {
			t.Fatalf("home's TV root holds %q, want the ten episodes", got)
		}
		simtest.Eventually(t, "the staging folder empties", func() bool { return len(c.staged(t)) == 0 })
		if n := transferRecords(t, c.w); n != 0 {
			t.Fatalf("home keeps %d Transfer records once every sent file is placed, want none", n)
		}
	})
}

// TestADeclineMidSendDeletesThatPart deletes The Copper Coast while an
// episode of a folder on seedbox is half sent into home's TV root. The
// file is declined work_deleted, and its staged .part goes.
func TestADeclineMidSendDeletesThatPart(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		p := newParked()
		c := launchCrossNode(t, func(e *node.Env) { e.Test.Transfers.Sent = p.sent })
		c.write(t, "pack/"+copperFolder+"/"+episodeName(1), 3*transfers.ChunkSize)
		id := c.open(t, "pack")
		c.await(t, id, "the episode waits confirm", func(v importView) bool { return v.count("waiting") == 1 })
		c.confirm(t, id)
		<-p.reached
		simtest.Eventually(t, "the first chunk is staged", func() bool { return slices.Equal(c.staged(t), []string{"0.part"}) })
		if r := c.w.API().Admin("home", http.MethodPost, "/api/v1/works/delete", `{"workIds":["`+c.work+`"],"files":"delete"}`); r.Status != http.StatusOK {
			t.Fatalf("deleteWorks = %d %s, want 200", r.Status, r.Body)
		}
		close(p.release)
		c.await(t, id, "the episode is declined", func(v importView) bool {
			return slices.Equal(v.outcomes(), []string{episodeName(1) + ": declined work_deleted"})
		})
		simtest.Eventually(t, "the .part is deleted", func() bool { return len(c.staged(t)) == 0 })
		if n := transferRecords(t, c.w); n != 0 {
			t.Fatalf("home keeps %d Transfer records once the file is declined, want none", n)
		}
	})
}
