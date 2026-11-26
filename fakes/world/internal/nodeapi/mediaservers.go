package nodeapi

import (
	"context"
	"net/http"
	"strconv"
)

const mediaServers = "/api/v1/media-servers"

// NewMediaServer is a Media Server as an Admin adds it.
type NewMediaServer struct {
	Kind            string `json:"kind"`
	Name            string `json:"name"`
	Address         string `json:"address,omitempty"`
	FallbackAddress string `json:"fallbackAddress,omitempty"`
	BrowserAddress  string `json:"browserAddress,omitempty"`
	ReachingNodeID  string `json:"reachingNodeId"`
	RunsOnNodeID    string `json:"runsOnNodeId,omitempty"`
	APIKey          string `json:"apiKey,omitempty"`
	DeviceID        string `json:"deviceId,omitempty"`
	PlexServer      string `json:"plexServer,omitempty"`
}

// AddMediaServer adds the server once both its routes answer as it, and
// returns its id.
func AddMediaServer(ctx context.Context, c *http.Client, baseURL, apiKey string, ms NewMediaServer) (string, error) {
	var created struct {
		ID string `json:"id"`
	}
	err := call(ctx, c, request{
		what: "add the " + ms.Kind + " Media Server", method: http.MethodPost, url: baseURL + mediaServers,
		header: apiKeyHeader(apiKey), body: ms, want: http.StatusCreated,
	}, &created)
	return created.ID, err
}

// SignedIn is the key and DeviceId a Jellyfin or Emby administrator
// sign-in created.
type SignedIn struct {
	APIKey   string `json:"apiKey"`
	DeviceID string `json:"deviceId"`
}

// SignInToMediaServer signs in to a Jellyfin or Emby as its administrator,
// from the Node with nodeID when the Controller doesn't reach it.
func SignInToMediaServer(ctx context.Context, c *http.Client, baseURL, apiKey, kind, address, nodeID, user, password string) (SignedIn, error) {
	var out SignedIn
	err := call(ctx, c, request{
		what: "sign in to the " + kind + " Media Server", method: http.MethodPost, url: baseURL + mediaServers + "/sign-in",
		header: apiKeyHeader(apiKey), body: map[string]string{"kind": kind, "address": address, "nodeId": nodeID, "user": user, "password": password},
		want: http.StatusOK,
	}, &out)
	return out, err
}

// MediaServerMapping is one library root's Path mapping.
type MediaServerMapping struct {
	RootID       string `json:"rootId"`
	Shown        bool   `json:"shown"`
	ServerFolder string `json:"serverFolder,omitempty"`
}

// MediaServerLibraries is every library root's mapping, pre-filled where
// it has none saved.
func MediaServerLibraries(ctx context.Context, c *http.Client, baseURL, apiKey, id string) ([]MediaServerMapping, error) {
	var out struct {
		Mappings []MediaServerMapping `json:"mappings"`
	}
	err := call(ctx, c, request{
		what: "list the Media Server's libraries", method: http.MethodGet, url: baseURL + mediaServers + "/" + id + "/libraries",
		header: apiKeyHeader(apiKey), want: http.StatusOK,
	}, &out)
	return out.Mappings, err
}

func SaveMediaServerMappings(ctx context.Context, c *http.Client, baseURL, apiKey, id string, mappings []MediaServerMapping) error {
	return call(ctx, c, request{
		what: "save the Media Server's Path mappings", method: http.MethodPut, url: baseURL + mediaServers + "/" + id + "/mappings",
		header: apiKeyHeader(apiKey), body: map[string]any{"mappings": mappings}, want: http.StatusOK,
	}, nil)
}

// PlexPin is a Plex sign-in waiting for the Admin on plex.tv.
type PlexPin struct {
	ID      int64  `json:"pinId"`
	AuthURL string `json:"authUrl"`
}

func StartPlexSignIn(ctx context.Context, c *http.Client, baseURL, apiKey, forward string) (PlexPin, error) {
	var out PlexPin
	err := call(ctx, c, request{
		what: "start the Plex sign-in", method: http.MethodPost, url: baseURL + mediaServers + "/plex/pin",
		header: apiKeyHeader(apiKey), body: map[string]string{"forward": forward}, want: http.StatusCreated,
	}, &out)
	return out, err
}

// PlexSignInState is waiting, expired or signed_in.
func PlexSignInState(ctx context.Context, c *http.Client, baseURL, apiKey string, id int64) (string, error) {
	var out struct {
		State string `json:"state"`
	}
	err := call(ctx, c, request{
		what: "read the Plex sign-in", method: http.MethodGet, url: baseURL + mediaServers + "/plex/pin/" + strconv.FormatInt(id, 10),
		header: apiKeyHeader(apiKey), want: http.StatusOK,
	}, &out)
	return out.State, err
}

// PlexServer is a server the signed-in Plex account owns.
type PlexServer struct {
	MachineIdentifier string `json:"machineIdentifier"`
	Name              string `json:"name"`
	Connections       []struct {
		URI string `json:"uri"`
	} `json:"connections"`
}

func ListPlexServers(ctx context.Context, c *http.Client, baseURL, apiKey string) ([]PlexServer, error) {
	var out struct {
		Servers []PlexServer `json:"servers"`
	}
	err := call(ctx, c, request{
		what: "list the Plex servers", method: http.MethodGet, url: baseURL + mediaServers + "/plex/servers",
		header: apiKeyHeader(apiKey), want: http.StatusOK,
	}, &out)
	return out.Servers, err
}

// Root is a library root as listRoots lists it.
type Root struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func ListRoots(ctx context.Context, c *http.Client, baseURL, apiKey string) ([]Root, error) {
	var out struct {
		Roots []Root `json:"roots"`
	}
	err := call(ctx, c, request{
		what: "list the library roots", method: http.MethodGet, url: baseURL + "/api/v1/roots",
		header: apiKeyHeader(apiKey), want: http.StatusOK,
	}, &out)
	return out.Roots, err
}

func cookieHeader(session *http.Cookie) http.Header {
	return http.Header{"Cookie": {session.String()}}
}

// ShowToMembers turns Show to Members on for server id, with the libraries
// behind its shown Path mappings.
func ShowToMembers(ctx context.Context, c *http.Client, baseURL, apiKey, id string) error {
	return call(ctx, c, request{
		what: "show Media Server " + id + " to Members", method: http.MethodPatch, url: baseURL + mediaServers + "/" + id,
		header: apiKeyHeader(apiKey), body: map[string]bool{"membersSee": true}, want: http.StatusOK,
	}, nil)
}

// AccessServer is a server a Member may be offered.
type AccessServer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AccessRow is one Member's access to one server.
type AccessRow struct {
	ID     string       `json:"id"`
	Server AccessServer `json:"server"`
	State  string       `json:"state"`
}

// Access is a Member's access rows and the servers they may be offered.
type Access struct {
	Rows      []AccessRow    `json:"rows"`
	Offerable []AccessServer `json:"offerable"`
}

func ListAccess(ctx context.Context, c *http.Client, baseURL, apiKey, account string) (Access, error) {
	var out Access
	err := call(ctx, c, request{
		what: "list the access of Account " + account, method: http.MethodGet, url: baseURL + mediaServers + "/access?account=" + account,
		header: apiKeyHeader(apiKey), want: http.StatusOK,
	}, &out)
	return out, err
}

// OfferMediaServer offers Account account the server and answers the
// access row's id.
func OfferMediaServer(ctx context.Context, c *http.Client, baseURL, apiKey, account, server string) (string, error) {
	var out struct {
		Row AccessRow `json:"row"`
	}
	err := call(ctx, c, request{
		what: "offer Media Server " + server + " to Account " + account, method: http.MethodPut, url: baseURL + "/api/v1/accounts/" + account + "/media-servers/" + server,
		header: apiKeyHeader(apiKey), body: struct{}{}, want: http.StatusOK,
	}, &out)
	return out.Row.ID, err
}

func StartMemberPlexSignIn(ctx context.Context, c *http.Client, baseURL string, session *http.Cookie) (PlexPin, error) {
	var out PlexPin
	err := call(ctx, c, request{
		what: "start the Member's Plex sign-in", method: http.MethodPost, url: baseURL + "/api/v1/account/media-servers/plex/pin",
		header: cookieHeader(session), want: http.StatusCreated,
	}, &out)
	return out, err
}

// MemberPlexSignIn is where a Member's Plex sign-in stands.
type MemberPlexSignIn struct {
	State   string `json:"state"`
	Refusal string `json:"refusal"`
}

func MemberPlexSignInState(ctx context.Context, c *http.Client, baseURL string, session *http.Cookie, id int64) (MemberPlexSignIn, error) {
	var out MemberPlexSignIn
	err := call(ctx, c, request{
		what: "read the Member's Plex sign-in", method: http.MethodGet, url: baseURL + "/api/v1/account/media-servers/plex/pin/" + strconv.FormatInt(id, 10),
		header: cookieHeader(session), want: http.StatusOK,
	}, &out)
	return out, err
}

// ShareAccess shares the Plex server of access row id as a friend, as the
// Admin of session.
func ShareAccess(ctx context.Context, c *http.Client, baseURL string, session *http.Cookie, id string) error {
	return call(ctx, c, request{
		what: "share access " + id, method: http.MethodPost, url: baseURL + mediaServers + "/access/" + id + "/share",
		header: cookieHeader(session), body: map[string]string{"as": "friend"}, want: http.StatusOK,
	}, nil)
}

// MakeMediaServerAccount makes the Member's user for access row id.
func MakeMediaServerAccount(ctx context.Context, c *http.Client, baseURL string, session *http.Cookie, id, username, password string) error {
	return call(ctx, c, request{
		what: "make the Member's user for access " + id, method: http.MethodPost, url: baseURL + "/api/v1/account/media-servers/" + id + "/account",
		header: cookieHeader(session), body: map[string]string{"username": username, "password": password}, want: http.StatusOK,
	}, nil)
}

func RemoveAccess(ctx context.Context, c *http.Client, baseURL, apiKey, id string) error {
	return call(ctx, c, request{
		what: "remove access " + id, method: http.MethodDelete, url: baseURL + mediaServers + "/access/" + id,
		header: apiKeyHeader(apiKey), want: http.StatusOK,
	}, nil)
}
