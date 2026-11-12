package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type folderImports struct {
	Imports []folderImport `json:"imports"`
}

type folderImport struct {
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

// lines is each file of ims as "base: outcome reason", sorted.
func (l folderImports) lines() []string {
	var out []string
	for _, im := range l.Imports {
		for _, f := range im.Files {
			line := filepath.Base(f.Path) + ": " + f.Outcome
			if f.Reason != nil {
				line += " " + f.Reason.Code
			}
			out = append(out, line)
		}
	}
	slices.Sort(out)
	return out
}

func (l folderImports) ids() []string {
	out := make([]string, len(l.Imports))
	for i, im := range l.Imports {
		out[i] = im.ID
	}
	slices.Sort(out)
	return out
}

// TestAFolderInADownloadFolderImportsIntoTwoWorksByHandAndStartsNoSearch
// grabs The Quiet Orchard through Transmission, which keeps seeding it,
// then opens Transmission's download folder by hand with two more copies of
// a video in it: one named for Harbor Lights and one no name matches.
// Reopening answers the same Imports, Move files is refused while the
// torrent seeds, a folder outside the fence is refused, and the unmatched
// file, matched to Salt Meadow, imports with Harbor Lights on Import all
// and no Grab follows.
func TestAFolderInADownloadFolderImportsIntoTwoWorksByHandAndStartsNoSearch(t *testing.T) {
	t.Parallel()
	s, err := scenario.Load("external-clients")
	if err != nil {
		t.Fatal(err)
	}
	s.Name = "external-clients-by-hand"
	s.TMDB.Movies = append(s.TMDB.Movies,
		scenario.TMDBMovie{ID: 9000002, IMDbID: "tt9000002", Title: "Harbor Lights", Released: "2019-09-06", Runtime: 98 * time.Minute},
		scenario.TMDBMovie{ID: 9000003, IMDbID: "tt9000003", Title: "Salt Meadow", Released: "2020-02-14", Runtime: 91 * time.Minute},
	)
	run := worldtest.LaunchScenario(t, s)
	m := run.Manifest()
	a := api(t, run)
	if g := endedGrab(t, a, grabbed(t, a, clientOrchard)); g.Stage != "imported" {
		t.Fatalf("the Orchard's Grab ended %s, want imported", g.Stage)
	}
	works := map[string]string{}
	for key, label := range map[string]string{"9000002": "Harbor Lights", "9000003": "Salt Meadow"} {
		w := decode(t, "addWork "+label, a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"`+key+`"}`), &added{})
		var tr tree
		decodeRaw(t, w.Work, &tr)
		works[label] = tr.ID
	}
	decode(t, "updateImportingSettings", a.Admin("home", http.MethodPatch, "/api/v1/imports/importing-settings", `{"values":{"library.importing.smallestVideo":0}}`), &struct{}{})

	folder := transmissionFolder(t, a)
	video := downloaded(t, folder, clientOrchard.name+".mkv")
	copyFile(t, video, filepath.Join(folder, "Harbor.Lights.2019.1080p.WEB-DL.H264-FKW.mkv"))
	copyFile(t, video, filepath.Join(folder, "Home Movie.mkv"))
	home := m.Nodes[0].NodeID
	open := func() worldtest.Reply {
		return a.Admin("home", http.MethodPost, "/api/v1/imports", `{"nodeId":"`+home+`","folder":`+strconv.Quote(folder)+`}`)
	}
	first := open()
	if first.Status != http.StatusCreated {
		t.Fatalf("importFolder = %d %s, want 201", first.Status, first.Body)
	}
	var opened folderImports
	decodeRaw(t, []byte(first.Body), &opened)
	if again := open(); again.Status != http.StatusOK || !slices.Equal(decodeRaw(t, []byte(again.Body), &folderImports{}).ids(), opened.ids()) {
		t.Fatalf("importFolder again = %d %s, want 200 with the same Imports %v", again.Status, again.Body, opened.ids())
	}
	worldtest.Expect(t, "importFolder outside", a.Admin("home", http.MethodPost, "/api/v1/imports", `{"nodeId":"`+home+`","folder":`+strconv.Quote(t.TempDir())+`}`),
		worldtest.Reply{Status: http.StatusUnprocessableEntity, Body: `{"status":422,"code":"import_outside","title":"The folder isn't in one of the Node's library roots or download folders"}` + "\n"})
	worldtest.Expect(t, "setImportMoveFiles", a.Admin("home", http.MethodPost, "/api/v1/imports/move-files", `{"importIds":["`+strings.Join(opened.ids(), `","`)+`"],"on":true}`),
		worldtest.Reply{Status: http.StatusConflict, Body: `{"status":409,"code":"import_move_seeding","title":"A tracked torrent seeds one of these files. Move files can't take a file while it seeds."}` + "\n"})

	read := func(what string, done func([]string) bool) folderImports {
		t.Helper()
		var (
			l    folderImports
			last []string
		)
		deadline := time.Now().Add(time.Minute)
		worldtest.Until(t, what, func() (bool, error) {
			l = folderImports{}
			decodeRaw(t, []byte(a.Admin("home", http.MethodGet, "/api/v1/imports?source=folder", "").Body), &l)
			last = l.lines()
			if !done(last) && time.Now().After(deadline) {
				return false, fmt.Errorf("the folder's files still read %q", last)
			}
			return done(last), nil
		})
		return l
	}
	l := read("the folder's files are read", func(lines []string) bool {
		return slices.Equal(lines, []string{
			"Harbor.Lights.2019.1080p.WEB-DL.H264-FKW.mkv: waiting confirm",
			"Home Movie.mkv: waiting no_match",
			clientOrchard.name + ".mkv: waiting not_an_upgrade",
		})
	})
	var meadow tree
	decodeRaw(t, []byte(a.Admin("home", http.MethodGet, "/api/v1/works/"+works["Salt Meadow"], "").Body), &meadow)
	meadowItem := meadow.Items[0].ID
	for _, im := range l.Imports {
		for _, f := range im.Files {
			if filepath.Base(f.Path) == "Home Movie.mkv" {
				body := `{"action":"match","workId":"` + works["Salt Meadow"] + `","itemIds":["` + meadowItem + `"],"edition":null,"releaseName":"Salt.Meadow.2020.1080p.WEB-DL.H264-FKW"}`
				decode(t, "actOnImportFile match", a.Admin("home", http.MethodPost, "/api/v1/imports/"+im.ID+"/files/"+strconv.Itoa(f.Position), body), &folderImport{})
			}
		}
	}
	l = read("the matched file waits confirm", func(lines []string) bool { return slices.Contains(lines, "Home Movie.mkv: waiting confirm") })
	decode(t, "confirmImports", a.Admin("home", http.MethodPost, "/api/v1/imports/confirm", `{"importIds":["`+strings.Join(l.ids(), `","`)+`"]}`), &folderImports{})
	read("Import all placed both and finished their Imports", func(lines []string) bool {
		return slices.Equal(lines, []string{clientOrchard.name + ".mkv: waiting not_an_upgrade"})
	})
	holdsOnly(t, m, harborMagnet.placed, meadowRedirect.placed, clientOrchard.placed)
	var grabs struct {
		Grabs []struct {
			WorkID string `json:"workId"`
		} `json:"grabs"`
	}
	decode(t, "listGrabs", a.Admin("home", http.MethodGet, "/api/v1/grabs", ""), &grabs)
	if len(grabs.Grabs) != 1 {
		t.Fatalf("listGrabs lists %d Grabs, want the Orchard's alone: the manual import started no search", len(grabs.Grabs))
	}
}

func decodeRaw[T any](t *testing.T, b []byte, v *T) *T {
	t.Helper()
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("decode %s: %v", b, err)
	}
	return v
}

// downloaded is the path of the file name under folder.
func downloaded(t *testing.T, folder, name string) string {
	t.Helper()
	var at string
	err := filepath.WalkDir(folder, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Name() == name {
			at = p
		}
		return err
	})
	if err != nil || at == "" {
		t.Fatalf("%s holds no %s: %v", folder, name, err)
	}
	return at
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	src, err := os.Open(from)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close() //nolint:errcheck // read only
	dst, err := os.Create(to)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatal(err)
	}
	if err := dst.Close(); err != nil {
		t.Fatal(err)
	}
}
