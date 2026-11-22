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
		a, err := d.conn.Advance(t.Context(), link.Job{Work: link.Carry{Changes: []link.JobChange{never, confirmed}}, Step: link.StepRestored})
		if err != nil || !a.Done {
			t.Fatalf("Advance from restored = %+v, %v, want done", a, err)
		}
		check(t, "outcomes", outcomes(a), []string{"accepted []", "accepted [watch_state collections]"})
		check(t, "refreshes", d.refreshes()[before:], []string{`folder C:\Media\Films\Nosferatu (1922)`})
	})
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
