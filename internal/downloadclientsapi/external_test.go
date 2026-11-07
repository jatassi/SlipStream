package downloadclientsapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// test is one Test the Node ran, on the Node at.
type test struct {
	at  role.NodeID
	req *dlv1.TestClientRequest
}

// calledOn is the Node's client as the Controller reaches it on at.
type calledOn struct {
	*node
	at role.NodeID
}

func (c calledOn) TestClient(_ context.Context, req *dlv1.TestClientRequest) (*dlv1.TestClientResponse, error) {
	c.tests = append(c.tests, test{at: c.at, req: req})
	if c.answer != nil {
		return c.answer, nil
	}
	return passing("4.1.1"), nil
}

func passing(version string) *dlv1.TestClientResponse {
	return &dlv1.TestClientResponse{Answer: &dlv1.TestClientResponse_Passed{Passed: &dlv1.Passed{Product: "Transmission", Version: version, NodeFolder: "/downloads"}}}
}

func failing(c dlv1.Cause) *dlv1.TestClientResponse {
	return &dlv1.TestClientResponse{Answer: &dlv1.TestClientResponse_Failed{Failed: &dlv1.Problem{Cause: c}}}
}

// downloader pairs a Node named name that holds Downloading.
func (w *world) downloader(t *testing.T, name string) role.NodeID {
	t.Helper()
	id := role.NodeID{UUID: storage.NewUUID()}
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, id, name)
	})
	if err != nil {
		t.Fatal(err)
	}
	w.roster.serving = append(w.roster.serving, id)
	return id
}

type clientView struct {
	ID             string `json:"id"`
	NodeID         string `json:"nodeId"`
	Name           string `json:"name"`
	Address        string `json:"address"`
	PasswordSaved  bool   `json:"passwordSaved"`
	On             bool   `json:"on"`
	Version        int64  `json:"settingsVersion"`
	Reach          string `json:"reach"`
	ProductVersion string `json:"productVersion"`
	Torrents       int    `json:"torrents"`
}

func transmissionOn(node role.NodeID) string {
	return `{"nodeId":"` + node.String() + `","kind":"transmission","address":"http://192.168.1.20:9091","username":"jackson","password":"hunter2"}`
}

// add creates a client from body, failing the test unless it is saved.
func (w *world) add(t *testing.T, body string) clientView {
	t.Helper()
	status, reply := w.o.Call(t, http.MethodPost, "/api/v1/external-download-clients", body)
	var c clientView
	if err := json.Unmarshal([]byte(reply), &c); status != http.StatusCreated || err != nil {
		t.Fatalf("POST %s = %d %s, want 201 and the client", body, status, reply)
	}
	return c
}

func (w *world) edit(t *testing.T, id, body string) clientView {
	t.Helper()
	status, reply := w.o.Call(t, http.MethodPatch, "/api/v1/external-download-clients/"+id, body)
	var c clientView
	if err := json.Unmarshal([]byte(reply), &c); status != http.StatusOK || err != nil {
		t.Fatalf("PATCH %s = %d %s, want 200 and the client", body, status, reply)
	}
	return c
}

type listed struct {
	Cards []struct {
		NodeID    string       `json:"nodeId"`
		GrabsTo   *string      `json:"grabsTo"`
		Externals []clientView `json:"externals"`
	} `json:"cards"`
}

func (w *world) externals(t *testing.T, node role.NodeID) []clientView {
	t.Helper()
	var l listed
	decodeReply(t, w, "/api/v1/download-clients", &l)
	for _, c := range l.Cards {
		if c.NodeID == node.String() {
			return c.Externals
		}
	}
	t.Fatalf("GET /download-clients has no card for %s", node)
	return nil
}

func TestCreateTestsOnTheNodeAndSavesOnlyOnAPass(t *testing.T) {
	w := serveWorld(t)
	w.node.answer = failing(dlv1.Cause_CAUSE_WRONG_CREDENTIALS)
	w.o.Expect(t, http.MethodPost, "/api/v1/external-download-clients", transmissionOn(w.self), http.StatusUnprocessableEntity,
		`{"status":422,"code":"download_client_test_failed","title":"The Download Client's Test failed",`+
			`"detail":"Transmission rejected the username or password. Check them and test again.","fields":["username","password"]}`)
	if got := w.externals(t, w.self); len(got) != 0 {
		t.Fatalf("after a failed Test the card lists %+v, want no client", got)
	}

	w.node.answer = nil
	c := w.add(t, transmissionOn(w.self))
	if c.Name != "Transmission" || c.Address != "http://192.168.1.20:9091/transmission/rpc" || !c.On || c.Version != 1 ||
		c.Reach != "responding" || c.ProductVersion != "4.1.1" || !c.PasswordSaved {
		t.Fatalf("the saved client = %+v, want Transmission at its usual path, on, at version 1, responding at 4.1.1 with its password saved", c)
	}
	if len(w.node.tests) != 2 {
		t.Fatalf("the Node ran %d Tests, want one for each create", len(w.node.tests))
	}
	sent := w.node.tests[1]
	if sent.at != w.self || sent.req.GetSecrets().GetPassword() != "hunter2" || sent.req.GetClient().GetUsername() != "jackson" ||
		sent.req.GetClient().GetLabel() != "slipstream" {
		t.Errorf("the Test ran on %s with %v, want the client's own Node, its password, username and the label slipstream", sent.at, sent.req)
	}
	if got := w.externals(t, w.self); len(got) != 1 || got[0].ID != c.ID {
		t.Errorf("the card lists %+v, want the saved client", got)
	}
}

func TestSecretsAreNeverInAResponse(t *testing.T) {
	w := serveWorld(t)
	_, created := w.o.Call(t, http.MethodPost, "/api/v1/external-download-clients", transmissionOn(w.self))
	var c clientView
	if err := json.Unmarshal([]byte(created), &c); err != nil {
		t.Fatal(err)
	}
	_, edited := w.o.Call(t, http.MethodPatch, "/api/v1/external-download-clients/"+c.ID, `{"password":"correct-horse"}`)
	_, tested := w.o.Call(t, http.MethodPost, "/api/v1/external-download-clients/"+c.ID+"/test", "")
	_, listed := w.o.Call(t, http.MethodGet, "/api/v1/download-clients", "")
	for _, reply := range []string{created, edited, tested, listed} {
		if strings.Contains(reply, "hunter2") || strings.Contains(reply, "correct-horse") {
			t.Errorf("a response holds a secret: %s", reply)
		}
	}
}

func TestAnEmptyPasswordKeepsTheSavedOne(t *testing.T) {
	w := serveWorld(t)
	c := w.add(t, transmissionOn(w.self))

	renamed := w.edit(t, c.ID, `{"name":"Living room","password":""}`)
	if len(w.node.tests) != 1 || renamed.Name != "Living room" || renamed.Version != 2 || renamed.Reach != "responding" || !renamed.PasswordSaved {
		t.Fatalf("renaming ran %d Tests and reads %+v, want no Test, version 2, still responding with the password saved", len(w.node.tests)-1, renamed)
	}

	w.edit(t, c.ID, `{"address":"http://192.168.1.21:9091","password":""}`)
	if got := w.node.tests[len(w.node.tests)-1].req.GetSecrets().GetPassword(); len(w.node.tests) != 2 || got != "hunter2" {
		t.Fatalf("a new address ran %d Tests with password %q, want one Test with the saved hunter2", len(w.node.tests)-1, got)
	}
	w.o.Call(t, http.MethodPost, "/api/v1/external-download-clients/"+c.ID+"/test", `{"username":"sam"}`)
	last := w.node.tests[len(w.node.tests)-1].req
	if last.GetSecrets().GetPassword() != "hunter2" || last.GetClient().GetUsername() != "sam" {
		t.Errorf("testing an unsaved username sent %v, want the saved password with the new username", last)
	}
	if got := w.externals(t, w.self)[0]; got.Version != 3 {
		t.Errorf("a Test of a saved client saved it at version %d, want it left at 3", got.Version)
	}
}

func TestATestedEditsTestIsThePollAtTheNewVersion(t *testing.T) {
	w := serveWorld(t)
	c := w.add(t, transmissionOn(w.self))
	w.node.answer = passing("4.1.2")
	got := w.edit(t, c.ID, `{"address":"http://192.168.1.21:9091"}`)
	if sent := w.node.tests[1].req.GetClient().GetSettingsVersion(); sent != 2 || got.Version != 2 || got.ProductVersion != "4.1.2" || got.Reach != "responding" {
		t.Errorf("the edit tested version %d and reads %+v, want the Test at version 2 and its 4.1.2 as the poll", sent, got)
	}

	w.node.answer = failing(dlv1.Cause_CAUSE_NO_RESPONSE)
	w.o.Expect(t, http.MethodPatch, "/api/v1/external-download-clients/"+c.ID, `{"address":"http://192.168.1.22:9091"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"download_client_test_failed","title":"The Download Client's Test failed",`+
			`"detail":"Transmission isn't responding. Check that it's running and that home can reach 192.168.1.22:9091.","fields":["address"]}`)
	if after := w.externals(t, w.self)[0]; after.Address != "http://192.168.1.21:9091/transmission/rpc" || after.Version != 2 {
		t.Errorf("after a failed Test the client reads %+v, want the last saved address at version 2", after)
	}
}

func TestAMoveIsTestedFromTheNewNodeAndRefusedWithTorrents(t *testing.T) {
	w := serveWorld(t)
	media := w.downloader(t, "Media server")
	c := w.add(t, transmissionOn(w.self))
	id := downloadclients.ClientID{UUID: must(storage.ParseUUID(c.ID))}
	torrent := w.holdIn(t, "Orchard", downloadclients.Target{Node: w.self, Client: &id}, nil)

	w.o.Expect(t, http.MethodPatch, "/api/v1/external-download-clients/"+c.ID, `{"nodeId":"`+media.String()+`"}`, http.StatusConflict,
		`{"status":409,"code":"download_client_has_torrents","title":"This client holds torrents SlipStream tracks",`+
			`"detail":"Transmission holds 1 torrent SlipStream tracks. Remove them, or delete the client with Stop tracking, then move it."}`)
	if len(w.node.tests) != 1 {
		t.Errorf("a refused move ran %d Tests, want none", len(w.node.tests)-1)
	}

	exec(t, w.lib, "UPDATE downloadclients_torrent SET removed_at = 1, removed_reason = 'admin', parted_at = 1, freed = 0 WHERE id = ?", torrent)
	moved := w.edit(t, c.ID, `{"nodeId":"`+media.String()+`"}`)
	if last := w.node.tests[len(w.node.tests)-1]; last.at != media || moved.NodeID != media.String() {
		t.Errorf("the move tested on %s and reads %+v, want it tested from Media server and saved there", last.at, moved)
	}
}

func TestGrabsToTakesOnlyAnOnClientOfItsNode(t *testing.T) {
	w := serveWorld(t)
	media := w.downloader(t, "Media server")
	settings := "/api/v1/download-clients/" + w.self.String() + "/settings"
	on := w.add(t, transmissionOn(w.self))
	off := w.add(t, `{"nodeId":"`+w.self.String()+`","kind":"deluge","address":"http://192.168.1.30:8112","password":"hunter2","on":false}`)
	elsewhere := w.add(t, `{"nodeId":"`+media.String()+`","kind":"transmission","name":"Attic","address":"http://192.168.1.40:9091"}`)

	w.o.Expect(t, http.MethodPatch, settings, `{"grabsTo":"`+off.ID+`"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"download_client_field_refused","title":"SlipStream can't use a value in this client's settings",`+
			`"detail":"Deluge is off. Turn it on before new Grabs go to it.","errors":[{"location":"body.grabsTo","message":"Deluge is off. Turn it on before new Grabs go to it."}]}`)
	w.o.Expect(t, http.MethodPatch, settings, `{"grabsTo":"`+elsewhere.ID+`"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"download_client_field_refused","title":"SlipStream can't use a value in this client's settings",`+
			`"detail":"Attic is on another Node. Choose a client on home.","errors":[{"location":"body.grabsTo","message":"Attic is on another Node. Choose a client on home."}]}`)

	if status, reply := w.o.Call(t, http.MethodPatch, settings, `{"grabsTo":"`+on.ID+`"}`); status != http.StatusOK || !strings.Contains(reply, `"grabsTo":"`+on.ID+`"`) {
		t.Fatalf("sending new Grabs to Transmission = %d %s, want 200 naming it", status, reply)
	}
	inUse := `{"status":409,"code":"download_client_in_use","title":"New Grabs go to this client",` +
		`"detail":"home's new Grabs go to this client. Choose another client for them first."}`
	w.o.Expect(t, http.MethodPatch, "/api/v1/external-download-clients/"+on.ID, `{"on":false}`, http.StatusConflict, inUse)
	w.o.Expect(t, http.MethodDelete, "/api/v1/external-download-clients/"+on.ID, "", http.StatusConflict, inUse)

	var target downloadclients.Target
	err := w.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		target, err = downloadclients.Route(ctx, r, downloadclients.Ask{Node: w.self})
		return err
	})
	if err != nil || target.Client == nil || target.Client.String() != on.ID {
		t.Fatalf("a Grab on home routes to %+v (%v), want Transmission", target, err)
	}

	if status, reply := w.o.Call(t, http.MethodPatch, settings, `{"grabsTo":null}`); status != http.StatusOK || !strings.Contains(reply, `"grabsTo":null`) {
		t.Fatalf("resetting new Grabs to the built-in client = %d %s, want 200", status, reply)
	}
	if status, _ := w.o.Call(t, http.MethodDelete, "/api/v1/external-download-clients/"+on.ID, ""); status != http.StatusNoContent {
		t.Fatalf("deleting Transmission once Grabs go elsewhere = %d, want 204", status)
	}
	if got := w.externals(t, w.self); len(got) != 1 || got[0].ID != off.ID {
		t.Errorf("after the delete home's card lists %+v, want Deluge alone", got)
	}
}

func TestDeleteRefusesAClientWithTorrents(t *testing.T) {
	w := serveWorld(t)
	c := w.add(t, transmissionOn(w.self))
	id := downloadclients.ClientID{UUID: must(storage.ParseUUID(c.ID))}
	w.holdIn(t, "Orchard", downloadclients.Target{Node: w.self, Client: &id}, nil)
	w.holdIn(t, "Harbor", downloadclients.Target{Node: w.self, Client: &id}, nil)
	w.o.Expect(t, http.MethodDelete, "/api/v1/external-download-clients/"+c.ID, "", http.StatusConflict,
		`{"status":409,"code":"download_client_has_torrents","title":"This client holds torrents SlipStream tracks",`+
			`"detail":"Transmission holds 2 torrents SlipStream tracks. Stop tracking them to delete it."}`)
	if got := w.externals(t, w.self); len(got) != 1 || got[0].Torrents != 2 {
		t.Errorf("after the refusal the card lists %+v, want Transmission with its 2 torrents", got)
	}
}

// TestDeleteWithStopTrackingLeavesTheTorrentsInTheClient deletes
// Transmission with Stop tracking: its torrents read gone with it, and
// home's intent neither lists nor discards them, so Transmission keeps
// them and their files.
func TestDeleteWithStopTrackingLeavesTheTorrentsInTheClient(t *testing.T) {
	w := serveWorld(t)
	c := w.add(t, transmissionOn(w.self))
	id := downloadclients.ClientID{UUID: must(storage.ParseUUID(c.ID))}
	orchard := w.holdIn(t, "Orchard", downloadclients.Target{Node: w.self, Client: &id}, nil)
	harbor := w.holdIn(t, "Harbor", downloadclients.Target{Node: w.self, Client: &id}, nil)

	if status, reply := w.o.Call(t, http.MethodDelete, "/api/v1/external-download-clients/"+c.ID+"?stopTracking=true", ""); status != http.StatusNoContent {
		t.Fatalf("deleting Transmission with Stop tracking = %d %s, want 204", status, reply)
	}
	if got := w.externals(t, w.self); len(got) != 0 {
		t.Errorf("after the delete home's card lists %+v, want no client", got)
	}
	var outcomes map[downloadclients.ID]downloadclients.Attached
	var intent *dlv1.Intent
	err := w.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if outcomes, err = downloadclients.Outcomes(ctx, r, []downloadclients.ID{orchard, harbor}); err != nil {
			return err
		}
		intent, err = downloadclients.Intent(ctx, r, w.self)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	gone := downloadclients.Gone{Reason: downloadclients.ClientDeleted, Client: "Transmission"}
	for _, torrent := range []downloadclients.ID{orchard, harbor} {
		if got := outcomes[torrent].Outcome; got != gone {
			t.Errorf("torrent %s reads %+v, want %+v", torrent, got, gone)
		}
	}
	if len(intent.GetTorrents()) != 0 || len(intent.GetDiscards()) != 0 {
		t.Errorf("home's intent lists %v and discards %v, want neither torrent", intent.GetTorrents(), intent.GetDiscards())
	}
}

func TestDeleteIsRefusedWhileAGrabImportsFromTheClient(t *testing.T) {
	w := serveWorld(t)
	c := w.add(t, transmissionOn(w.self))
	id := downloadclients.ClientID{UUID: must(storage.ParseUUID(c.ID))}
	orchard := w.holdIn(t, "Orchard", downloadclients.Target{Node: w.self, Client: &id}, nil)
	w.uses[orchard] = []downloadclients.Use{{Grab: storage.NewUUID(), Release: orchardRelease, Stage: downloadclients.UseImporting}}

	w.o.Expect(t, http.MethodDelete, "/api/v1/external-download-clients/"+c.ID+"?stopTracking=true", "", http.StatusConflict,
		`{"status":409,"code":"download_client_importing","title":"A Grab is importing from this client",`+
			`"detail":"A Grab is importing from Transmission. Delete it once the import finishes."}`)
	if got := w.externals(t, w.self); len(got) != 1 || got[0].Torrents != 1 {
		t.Errorf("after the refusal the card lists %+v, want Transmission with its torrent", got)
	}
}

func TestClientsAreRefusedAtTheBoundary(t *testing.T) {
	w := serveWorld(t)
	w.add(t, transmissionOn(w.self))
	for _, c := range []struct {
		name, body, want string
		status           int
	}{
		{
			"sign-in in the address",
			`{"nodeId":"` + w.self.String() + `","kind":"transmission","name":"Other","address":"http://jackson:hunter2@192.168.1.21:9091"}`,
			`{"status":422,"code":"download_client_field_refused","title":"SlipStream can't use a value in this client's settings",` +
				`"detail":"This address holds a username or password. Put them in the Username and Password fields.",` +
				`"errors":[{"location":"body.address","message":"This address holds a username or password. Put them in the Username and Password fields."}]}`,
			http.StatusUnprocessableEntity,
		},
		{
			"a comma in a Transmission label",
			`{"nodeId":"` + w.self.String() + `","kind":"transmission","name":"Other","address":"http://192.168.1.21:9091","label":"tv,movies"}`,
			`{"status":422,"code":"download_client_field_refused","title":"SlipStream can't use a value in this client's settings",` +
				`"detail":"Transmission splits labels at a comma. Use a label without one.",` +
				`"errors":[{"location":"body.label","message":"Transmission splits labels at a comma. Use a label without one."}]}`,
			http.StatusUnprocessableEntity,
		},
		{
			"a Deluge with no password",
			`{"nodeId":"` + w.self.String() + `","kind":"deluge","address":"http://192.168.1.30:8112"}`,
			`{"status":422,"code":"download_client_field_refused","title":"SlipStream can't use a value in this client's settings",` +
				`"detail":"Enter Deluge's password.","errors":[{"location":"body.password","message":"Enter Deluge's password."}]}`,
			http.StatusUnprocessableEntity,
		},
		{
			"the same address",
			`{"nodeId":"` + w.self.String() + `","kind":"transmission","name":"Other","address":"HTTP://192.168.1.20:9091/transmission/rpc/"}`,
			`{"status":409,"code":"download_client_address_taken","title":"Another Download Client already uses this address",` +
				`"detail":"Transmission on home already uses this address."}`,
			http.StatusConflict,
		},
		{
			"the same name",
			`{"nodeId":"` + w.self.String() + `","kind":"transmission","address":"http://192.168.1.21:9091"}`,
			`{"status":409,"code":"download_client_name_taken","title":"Another Download Client has that name"}`,
			http.StatusConflict,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			w.o.Expect(t, http.MethodPost, "/api/v1/external-download-clients", c.body, c.status, c.want)
		})
	}
	if len(w.node.tests) != 1 {
		t.Errorf("refused drafts ran %d Tests, want none", len(w.node.tests)-1)
	}
}

func TestATestOnAnOfflineNodeCouldNotRun(t *testing.T) {
	w := serveWorld(t)
	w.roster.away[w.self] = true
	w.o.Expect(t, http.MethodPost, "/api/v1/external-download-clients/test", transmissionOn(w.self), http.StatusBadGateway,
		`{"status":502,"code":"download_client_unavailable","title":"The Test couldn't run",`+
			`"detail":"SlipStream couldn't reach home to run the Test. Test again once home is back."}`)
	w.o.Expect(t, http.MethodPost, "/api/v1/external-download-clients/test", `{"nodeId":"0b0e3b9c-7c0a-4c8e-9a51-2f7d1e7f9a10","kind":"transmission","address":"http://192.168.1.20:9091"}`,
		http.StatusConflict, `{"status":409,"code":"node_not_downloading","title":"That Node doesn't hold Downloading"}`)
}

func TestADraftTestAnswersTheVersionAndWarnings(t *testing.T) {
	w := serveWorld(t)
	w.node.answer = &dlv1.TestClientResponse{Answer: &dlv1.TestClientResponse_Passed{Passed: &dlv1.Passed{
		Product: "qBittorrent", Version: "5.1.0", NodeFolder: `D:\Torrents`, Warnings: []dlv1.Warning{dlv1.Warning_WARNING_REMOVES_AT_LIMIT},
	}}}
	w.o.Expect(t, http.MethodPost, "/api/v1/external-download-clients/test",
		`{"nodeId":"`+w.self.String()+`","kind":"qbittorrent","address":"http://192.168.1.20:8080","apiKey":"qbt_key"}`, http.StatusOK,
		`{"product":"qBittorrent","version":"5.1.0","nodeFolder":"D:\\Torrents","warnings":["qBittorrent removes torrents that reach a seeding limit. `+
			`A download could be removed before SlipStream imports it. To keep downloads until they're imported, set qBittorrent to stop torrents at the limit."],"daemons":[]}`)
	if got := w.externals(t, w.self); len(got) != 0 {
		t.Errorf("a draft's Test saved %+v, want nothing saved", got)
	}
}

func exec(t *testing.T, lib *storage.DB[storage.Library], query string, args ...any) {
	t.Helper()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestADelugeTestThatNeedsADaemonListsTheDaemons(t *testing.T) {
	w := serveWorld(t)
	deluge := `{"nodeId":"` + w.self.String() + `","kind":"deluge","address":"http://192.168.1.20:8112","password":"hunter2"}`
	w.node.answer = &dlv1.TestClientResponse{Answer: &dlv1.TestClientResponse_Failed{Failed: &dlv1.Problem{
		Cause: dlv1.Cause_CAUSE_DAEMON_NEEDED, Daemons: []*dlv1.Daemon{{Host: "127.0.0.1", Port: 58846}, {Host: "192.168.1.30", Port: 58846}},
	}}}
	w.o.Expect(t, http.MethodPost, "/api/v1/external-download-clients/test", deluge, http.StatusUnprocessableEntity,
		`{"status":422,"code":"download_client_choose_daemon","title":"Choose the daemon SlipStream uses",`+
			`"detail":"Deluge's web UI lists more than one daemon. Choose the daemon SlipStream uses and test again.",`+
			`"daemons":[{"host":"127.0.0.1","port":58846},{"host":"192.168.1.30","port":58846}]}`)

	w.node.answer = failing(dlv1.Cause_CAUSE_WRONG_CREDENTIALS)
	w.o.Expect(t, http.MethodPost, "/api/v1/external-download-clients/test", deluge, http.StatusUnprocessableEntity,
		`{"status":422,"code":"download_client_test_failed","title":"The Download Client's Test failed",`+
			`"detail":"Deluge rejected the password. Check it and test again.","fields":["password"]}`)
}
