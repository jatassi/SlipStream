package mediaserversapi_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaserversapi"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// notYet is every operation still answering 501. A PR that builds one
// deletes its line here and its method on mediaservers.Unbuilt.
var notYet = []string{
	"addMediaServer",
	"updateMediaServer",
	"removeMediaServer",
	"testMediaServer",
	"confirmMediaServerIdentity",
	"listMediaServerLibraries",
	"saveMediaServerMappings",
	"refreshMediaServer",
	"listMediaServerSessions",
	"listMediaServerChanges",
	"listMediaServerCredentialHolders",
	"signInToMediaServer",
	"startPlexSignIn",
	"getPlexSignIn",
	"getPlexAccount",
	"signOutOfPlex",
	"listPlexServers",
}

const (
	server = "/api/v1/media-servers/0b6f3a8e-2c4d-4e1f-9a7b-5d8c6e2f1a3b"
	node   = "5c1d7e9a-3b2f-4a6c-8d0e-1f4a7b9c2e5d"
)

type call struct{ method, path, body string }

var calls = map[string]call{
	"listMediaServerKinds":             {http.MethodGet, "/api/v1/media-servers/kinds", ""},
	"listMediaServers":                 {http.MethodGet, "/api/v1/media-servers", ""},
	"addMediaServer":                   {http.MethodPost, "/api/v1/media-servers", `{"kind":"jellyfin","name":"Den Jellyfin","address":"http://192.168.1.20:8096","reachingNodeId":"` + node + `","apiKey":"k"}`},
	"getMediaServer":                   {http.MethodGet, server, ""},
	"updateMediaServer":                {http.MethodPatch, server, `{"name":"Living room Jellyfin"}`},
	"removeMediaServer":                {http.MethodDelete, server, ""},
	"testMediaServer":                  {http.MethodPost, server + "/test", ""},
	"confirmMediaServerIdentity":       {http.MethodPost, server + "/confirm-identity", ""},
	"listMediaServerLibraries":         {http.MethodGet, server + "/libraries", ""},
	"saveMediaServerMappings":          {http.MethodPut, server + "/mappings", `{"mappings":[{"rootId":"` + node + `","shown":false}]}`},
	"refreshMediaServer":               {http.MethodPost, server + "/refresh", ""},
	"listMediaServerSessions":          {http.MethodGet, server + "/sessions", ""},
	"listMediaServerChanges":           {http.MethodGet, server + "/changes", ""},
	"listMediaServerCredentialHolders": {http.MethodGet, server + "/credential-holders", ""},
	"signInToMediaServer":              {http.MethodPost, "/api/v1/media-servers/sign-in", `{"kind":"emby","address":"http://192.168.1.20:8096","nodeId":"` + node + `","user":"ada","password":"p"}`},
	"startPlexSignIn":                  {http.MethodPost, "/api/v1/media-servers/plex/pin", `{"forward":"https://slipstream.example.test/settings/media-servers"}`},
	"getPlexSignIn":                    {http.MethodGet, "/api/v1/media-servers/plex/pin/7", ""},
	"getPlexAccount":                   {http.MethodGet, "/api/v1/media-servers/plex/account", ""},
	"signOutOfPlex":                    {http.MethodDelete, "/api/v1/media-servers/plex/account", ""},
	"listPlexServers":                  {http.MethodGet, "/api/v1/media-servers/plex/servers", ""},
}

func serve(t *testing.T) *ownertest.Owner {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema())
	return ownertest.Serve(t, lib, func(r *api.Router) { mediaserversapi.Routes(r, mediaservers.Unbuilt{}) })
}

func documentedOps(t *testing.T) []string {
	t.Helper()
	var doc struct {
		Paths map[string]map[string]struct {
			ID   string   `json:"operationId"`
			Tags []string `json:"tags"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(surface.Document(), &doc); err != nil {
		t.Fatal(err)
	}
	var ops []string
	for _, methods := range doc.Paths {
		for _, op := range methods {
			if slices.Contains(op.Tags, "mediaservers") {
				ops = append(ops, op.ID)
			}
		}
	}
	slices.Sort(ops)
	return ops
}

func TestNotYetLedger(t *testing.T) {
	o := serve(t)
	ops := documentedOps(t)
	var called []string
	for id := range calls {
		called = append(called, id)
	}
	slices.Sort(called)
	if !slices.Equal(ops, called) {
		t.Fatalf("the document's mediaservers operations are %v, and the test calls %v", ops, called)
	}
	for _, id := range ops {
		c := calls[id]
		status, body := o.Call(t, c.method, c.path, c.body)
		built, listed := status != http.StatusNotImplemented, slices.Contains(notYet, id)
		if built == listed {
			t.Errorf("%s answered %d %s, and the ledger lists it: %t", id, status, body, listed)
		}
		if !built && !strings.Contains(body, `"code":"media_servers_not_built"`) {
			t.Errorf("%s answered 501 with %s", id, body)
		}
	}
}

func TestUnbuiltListsNoServersAndFindsNone(t *testing.T) {
	o := serve(t)
	o.Expect(t, http.MethodGet, "/api/v1/media-servers", "", http.StatusOK, `{"servers":[]}`)
	o.Expect(t, http.MethodGet, server, "", http.StatusNotFound,
		`{"status":404,"code":"media_server_not_found","title":"No Media Server has this id"}`)
}
