package plextv

import "github.com/jatassi/slipstream/internal/node/endpoint"

// call names one plex.tv call SlipStream makes.
type call int

const (
	pin call = iota
	pollPin
	user
	resources
	authNonce
	authToken
	legacyPin
	pollLegacyPin
	signOut
)

type format int

const (
	formatJSON format = iota
	formatXML
)

// route is one plex.tv call as a recording found it. Its path names each
// part a call fills in as {name}, and may carry a fixed query.
type route struct {
	host   func(Endpoints) endpoint.URL
	method string
	path   string
	format format
}

func api(e Endpoints) endpoint.URL { return e.API }

var routes = map[call]route{
	// The Admin's device. Measured by the JWT run of 2026-11-14.
	pin:       {api, "POST", "/api/v2/pins", formatJSON},
	pollPin:   {api, "GET", "/api/v2/pins/{pin}", formatJSON},
	user:      {api, "GET", "/api/v2/user", formatJSON},
	resources: {api, "GET", "/api/v2/resources", formatJSON},
	authNonce: {api, "GET", "/api/v2/auth/nonce", formatJSON},
	authToken: {api, "POST", "/api/v2/auth/token", formatJSON},
	// A Member's device. Measured by the Mac's signin.jsonl (#74) and by the
	// Plex run of 2026-11-25 (q3), which got a 20-character token from a
	// strong PIN with no key on clients.plex.tv and on plex.tv. The Member's
	// own /api/v2/user is the user route above.
	legacyPin:     {api, "POST", "/api/v2/pins?strong=true", formatJSON},
	pollLegacyPin: {api, "GET", "/api/v2/pins/{pin}", formatJSON},
	signOut:       {api, "DELETE", "/api/v2/users/signout", formatJSON},
}
