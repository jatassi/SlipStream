package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/emby"
	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/fakes/plex"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

const orchardFile = "The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv"

type (
	idPath struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	idName struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	reportedPath struct {
		Path       string `json:"path"`
		UpdateType string `json:"updateType"`
	}
	pathReport struct {
		Updates []reportedPath `json:"updates"`
	}
)

type mediaChange struct {
	Kind  string `json:"kind"`
	Path  string `json:"path"`
	State string `json:"state"`
	Words string `json:"words"`
}

// importedInto launches the media-servers Scenario, adds The Quiet
// Orchard into the root at folder and searches it until its Grab is
// imported.
func importedInto(t *testing.T, folder string) walked {
	t.Helper()
	s, err := scenario.Load("media-servers")
	if err != nil {
		t.Fatal(err)
	}
	w := walked{s: s, run: worldtest.LaunchScenario(t, s)}
	w.m, w.a = w.run.Manifest(), api(t, w.run)
	// An import while the Path mappings' first whole library refresh still
	// waits is covered by it, and no folder refresh or path report follows.
	for _, srv := range mediaServers(t, w) {
		worldtest.Until(t, srv.Name+" has taken its first whole library refresh", func() (bool, error) {
			changes := serverChanges(t, w, srv.ID)
			return len(changes) > 0 && !slices.ContainsFunc(changes, func(c mediaChange) bool { return c.State != "sent" }), nil
		})
	}
	var roots struct {
		Roots []idPath `json:"roots"`
	}
	decode(t, "listRoots", w.a.Admin("home", http.MethodGet, "/api/v1/roots", ""), &roots)
	i := slices.IndexFunc(roots.Roots, func(r idPath) bool { return filepath.Base(r.Path) == folder })
	if i < 0 {
		t.Fatalf("listRoots = %+v, want a root at %s", roots.Roots, folder)
	}
	first := decode(t, "addWork", w.a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9000001","preset":"all","rootId":"`+roots.Roots[i].ID+`"}`), &added{})
	var tr tree
	decodeRaw(t, first.Work, &tr)
	w.work = tr.ID
	w.search(t)
	var grabs struct {
		Grabs []grabView `json:"grabs"`
	}
	worldtest.Until(t, "the search grabs a release", func() (bool, error) {
		decode(t, "listGrabs", w.a.Admin("home", http.MethodGet, "/api/v1/grabs", ""), &grabs)
		return len(grabs.Grabs) == 1, nil
	})
	w.ended(t, grabs.Grabs[0].ID)
	if w.grab.Stage != "imported" {
		t.Fatalf("the Grab ended %s with %+v, want imported", w.grab.Stage, w.grab.Failure)
	}
	return w
}

func mediaServers(t *testing.T, w walked) []idName {
	t.Helper()
	var servers struct {
		Servers []idName `json:"servers"`
	}
	return decode(t, "listMediaServers", w.a.Admin("home", http.MethodGet, "/api/v1/media-servers", ""), &servers).Servers
}

func serverChanges(t *testing.T, w walked, id string) []mediaChange {
	t.Helper()
	var l struct {
		Changes []mediaChange `json:"changes"`
	}
	return decode(t, "listMediaServerChanges", w.a.Admin("home", http.MethodGet, "/api/v1/media-servers/"+id+"/changes", ""), &l).Changes
}

// sentAdded waits until the server named name lists the imported file's
// change as sent, and answers its words.
func sentAdded(t *testing.T, w walked, name string) string {
	t.Helper()
	servers := mediaServers(t, w)
	i := slices.IndexFunc(servers, func(s idName) bool { return s.Name == name })
	if i < 0 {
		t.Fatalf("listMediaServers = %+v, want %s", servers, name)
	}
	var sent mediaChange
	worldtest.Until(t, name+" lists the imported file's change as sent", func() (bool, error) {
		for _, c := range serverChanges(t, w, servers[i].ID) {
			if c.Kind == "added" && c.Path == orchardFile && c.State == "sent" {
				sent = c
				return true, nil
			}
		}
		return false, nil
	})
	return sent.Words
}

// captured waits until the run's capture of kind holds a record from
// node, any Node when empty, whose data matches.
func captured[T any](t *testing.T, w walked, kind, node string, match func(T) bool) {
	t.Helper()
	worldtest.Until(t, fmt.Sprintf("%s records %s", node, kind), func() (bool, error) {
		recs, err := events.Select(w.m.RunDir, events.Query{Kind: kind, Node: node})
		for _, r := range recs {
			var v T
			if json.Unmarshal(r.Data, &v) == nil && match(v) {
				return true, err
			}
		}
		return false, err
	})
}

type serverItem struct {
	Event string   `json:"event"`
	Files []string `json:"files"`
}

func addedItem(file string) func(serverItem) bool {
	return func(it serverItem) bool { return it.Event == "added" && slices.Contains(it.Files, file) }
}

// TestAnImportIntoHomesRootReachesPlex imports The Quiet Orchard into
// home's movies root, which Living room Plex shows: home refreshes the
// movie's folder on Plex, Plex adds the item, and the change reads sent.
func TestAnImportIntoHomesRootReachesPlex(t *testing.T) {
	t.Parallel()
	w := importedInto(t, "movies")
	const file = "/media/movies/" + orchardFile
	captured(t, w, plex.Refreshes.Kind, "home", func(r plex.Refresh) bool { return r.Path == path.Dir(file) })
	captured(t, w, plex.Items.Kind, "", addedItem(file))
	if words := sentAdded(t, w, "Living room Plex"); words != "Sent the new file." {
		t.Fatalf("Living room Plex's change reads %q, want Sent the new file.", words)
	}
}

// TestAnImportIntoDensRootReachesJellyfinAndEmby imports The Quiet Orchard
// into den's den-movies root, which Den Jellyfin and Den Emby show: den
// reports the new file's path to each, each adds the item, and each
// change reads sent.
func TestAnImportIntoDensRootReachesJellyfinAndEmby(t *testing.T) {
	t.Parallel()
	w := importedInto(t, "den-movies")
	const file = "/media/den-movies/" + orchardFile
	created := func(r pathReport) bool {
		return slices.Contains(r.Updates, reportedPath{Path: file, UpdateType: "Created"})
	}
	for name, kinds := range map[string][2]string{
		"Den Jellyfin": {jellyfin.Reports.Kind, jellyfin.Items.Kind},
		"Den Emby":     {emby.Reports.Kind, emby.Items.Kind},
	} {
		captured(t, w, kinds[0], "den", created)
		captured(t, w, kinds[1], "", addedItem(file))
		if words := sentAdded(t, w, name); words != "Sent the new file." {
			t.Fatalf("%s's change reads %q, want Sent the new file.", name, words)
		}
	}
}
