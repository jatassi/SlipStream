package plexkind_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/internal/plexkind"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
	"github.com/jatassi/slipstream/internal/storage"
)

const metropolis = "Films/Metropolis (1927)/Metropolis (1927).mkv"

// moveChange is a Move's carry from Films to Vault.
func moveChange(to, from link.ServerPath) link.JobChange {
	return link.JobChange{ID: link.ChangeID{UUID: storage.NewUUID()}, Kind: link.ChangeMoved, Library: "5", Path: to, From: from, FromLibrary: "1"}
}

func TestAMoveNeverConfirmedRefreshesItsDestinationAgainAndNeverPasses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const nosferatu = "Films/Nosferatu (1922)/Nosferatu (1922).mkv"
		d := newDen(t, nosferatu)
		d.watched(nosferatu)
		dest := d.at("Vault/Nosferatu (1922)/Nosferatu (1922).mkv")
		j := link.Job{Work: link.Carry{Changes: []link.JobChange{moveChange(dest, d.at(nosferatu))}}, Step: link.StepNone}
		start := time.Now()
		for time.Since(start) <= 2*link.DefaultTimings().MovedConfirm {
			a, err := d.conn.Advance(t.Context(), j)
			if err != nil {
				t.Fatalf("Advance at %s: %v", j.Step, err)
			}
			if a.Done || a.Job.Step == link.StepConfirmed {
				t.Fatalf("the Move's carry passed %s with no item at its new path", a.Job.Step)
			}
			time.Sleep(time.Until(a.Wait))
			j = a.Job
		}
		check(t, "refreshes", d.refreshes(), []string{`folder C:\Media\Vault\Nosferatu (1922)`, `folder C:\Media\Vault\Nosferatu (1922)`, `folder C:\Media\Vault\Nosferatu (1922)`})
	})
}

func TestAMoveUnconfirmedOnceItsSourceIsGoneEndsLostAndLeavesItsSourceFolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const nosferatu = "Films/Nosferatu (1922)/Nosferatu (1922).mkv"
		d := newDen(t, metropolis, nosferatu, "Vault/Faust (1926)/Faust (1926).mkv")
		d.watched(metropolis)
		lostKey, _ := d.item(metropolis)
		d.place("Vault/Nosferatu (1922)/Nosferatu (1922).mkv")
		copied := d.at("Vault/Nosferatu (1922)/Nosferatu (1922).mkv")
		d.drive(link.Job{Work: link.Refresh{Changes: []link.JobChange{{ID: link.ChangeID{UUID: storage.NewUUID()}, Kind: link.ChangeAdded, Library: "5", Path: copied}}}, Step: link.StepNone})
		keptKey, _ := d.item(nosferatu)
		d.remove(metropolis)
		d.remove(nosferatu)
		before := len(d.refreshes())
		never := moveChange(d.at("Vault/Metropolis (1927)/Metropolis (1927).mkv"), d.at(metropolis))
		never.Snapshot = &link.Snapshot{Item: lostKey}
		confirmed := moveChange(copied, d.at(nosferatu))
		confirmed.Snapshot = &link.Snapshot{Item: keptKey}
		_, a := d.drive(link.Job{Work: link.Carry{Changes: []link.JobChange{never, confirmed}}, Step: link.StepRestored})
		check(t, "outcomes", outcomes(a), []string{"accepted []", "accepted [watch_state collections]"})
		check(t, "refreshes", d.refreshes()[before:], []string{`folder C:\Media\Films\Nosferatu (1922)`})
	})
}

// TestAPlexMoveKeepsTheItemsWatchStateRatingAndCollections drives a
// Move's carry as the Move runs it: copy, steps to the hold, delete, the
// rest. Vault holds another film, since Plex ignores a partial refresh of
// a section never scanned with content. Plex's own scan of the vacated
// folder can land before SlipStream's refresh (recorded run 2026-11-23,
// case 4).
func TestAPlexMoveKeepsTheItemsWatchStateRatingAndCollections(t *testing.T) {
	for _, c := range []struct {
		name, from, to, fromLibrary string
		plexScansFirst              bool
	}{
		{name: "another section", from: metropolis, to: "Vault/Metropolis (1927)/Metropolis (1927).mkv", fromLibrary: "1"},
		{name: "its section", from: "Archive/Metropolis (1927)/Metropolis (1927).mkv", to: "Vault/Metropolis (1927)/Metropolis (1927).mkv", fromLibrary: "5"},
		{name: "its section, Plex scanning first", from: "Archive/Metropolis (1927)/Metropolis (1927).mkv", to: "Vault/Metropolis (1927)/Metropolis (1927).mkv", fromLibrary: "5", plexScansFirst: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				d := newDen(t, c.from, "Vault/Faust (1926)/Faust (1926).mkv")
				d.watched(c.from)
				key, _ := d.item(c.from)
				d.send(http.MethodPut, "/:/rate", url.Values{"key": {key.Item}, "identifier": {"com.plexapp.plugins.library"}, "rating": {"8"}})
				d.place(c.to)
				ch := link.JobChange{ID: link.ChangeID{UUID: storage.NewUUID()}, Kind: link.ChangeMoved, Library: "5", Path: d.at(c.to), From: d.at(c.from), FromLibrary: c.fromLibrary}
				j := d.toHold(link.Job{Work: link.Carry{Changes: []link.JobChange{ch}}, Step: link.StepNone})
				d.remove(c.from)
				if c.plexScansFirst {
					d.send(http.MethodGet, "/library/sections/"+c.fromLibrary+"/refresh", url.Values{"path": {d.at(c.from).Dir().String()}})
				}
				_, a := d.drive(j)
				check(t, "outcomes", outcomes(a), []string{"accepted []"})
				if _, ok := d.item(c.from); ok {
					t.Errorf("an item is still at %s", c.from)
				}
				if m := d.shown(c.to); m.ViewCount != 1 || m.UserRating != 8 || len(m.Collections) != 1 || m.Collections[0].Tag != "Restored" {
					t.Errorf("the item at %s has play count %d, rating %v and collections %v, want 1, 8 and Restored", c.to, m.ViewCount, m.UserRating, m.Collections)
				}
			})
		})
	}
}

// toHold advances a Move's carry to the step it holds at until the Move
// removed its source.
func (d *den) toHold(j link.Job) link.Job {
	d.t.Helper()
	for j.Step != link.StepRestored {
		a, err := d.conn.Advance(d.t.Context(), j)
		if err != nil || a.Done {
			d.t.Fatalf("Advance at %s = %+v, %v, want a step to restored", j.Step, a, err)
		}
		time.Sleep(time.Until(a.Wait))
		j = a.Job
	}
	return j
}

// shown is the item at rel as Plex answers it.
func (d *den) shown(rel string) plex.Metadata {
	d.t.Helper()
	k, ok := d.item(rel)
	if !ok {
		d.t.Fatalf("no item at %s", rel)
	}
	base, err := url.Parse(d.f.Address.URL(""))
	if err != nil {
		d.t.Fatal(err)
	}
	m, _, err := plex.New(base, token, plex.Headers{ClientID: "slipstream-test"}, d.f.Client).Item(d.t.Context(), plex.ID(k.Item))
	if err != nil {
		d.t.Fatal(err)
	}
	return m
}

// recorder keeps the query of every item edit it carries.
type recorder struct {
	base  http.RoundTripper
	edits *[]url.Values
}

func (r recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodPut && strings.HasPrefix(req.URL.Path, "/library/metadata/") && !strings.HasSuffix(req.URL.Path, "/refresh") {
		*r.edits = append(*r.edits, req.URL.Query())
	}
	resp, err := r.base.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("record %s: %w", req.URL.Path, err)
	}
	return resp, nil
}

func TestRestoreCollectionsSendsTheItemsOwnCollectionsWithTheSnapshots(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const vault = "Vault/Metropolis (1927)/Metropolis (1927).mkv"
		d := newDen(t, vault)
		dest := d.at(vault)
		d.drive(link.Job{Work: link.Refresh{Changes: []link.JobChange{{ID: link.ChangeID{UUID: storage.NewUUID()}, Kind: link.ChangeAdded, Library: "5", Path: dest}}}, Step: link.StepNone})
		key, _ := d.item(vault)
		d.send(http.MethodPut, "/library/metadata/"+key.Item, url.Values{"collection[0].tag.tag": {"Silents"}})
		var edits []url.Values
		conn := plexkind.Kind.Dial(link.Dialed{
			Server:  link.ServerRef{ID: link.ServerID{UUID: storage.NewUUID()}, Kind: "plex", Identity: machine},
			Address: d.f.Address.URL(""), Credential: link.Credential{Token: token}, DeviceID: "slipstream-test",
			HTTP: &http.Client{Transport: recorder{base: d.f.Client.Transport, edits: &edits}}, Timings: link.DefaultTimings(),
		})
		ch := moveChange(dest, d.at(metropolis))
		ch.Snapshot = &link.Snapshot{Item: link.ItemKey{Item: "999"}, Collections: []string{"Silents", "Expressionism"}}
		a, err := conn.Advance(t.Context(), link.Job{Work: link.Carry{Changes: []link.JobChange{ch}}, Step: link.StepConfirmed})
		if err != nil || a.Job.Step != link.StepRestored {
			t.Fatalf("Advance from confirmed = %+v, %v, want restored", a, err)
		}
		check(t, "edits", edits, []url.Values{{
			"type": {"1"}, "id": {key.Item}, "collection.locked": {"1"},
			"collection[0].tag.tag": {"Silents"}, "collection[1].tag.tag": {"Expressionism"},
		}})
	})
}
