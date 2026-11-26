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
	sections
	ownedAccepted
	ownedPending
	share
	deleteShare
	invites
	cancelInvite
	homeUsers
	inviteHome
	leaveHome
	pendingReceived
	accept
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

func web(e Endpoints) endpoint.URL { return e.Web }

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
	// The Admin's JWT on the sharing routes, measured 2026-11-25
	// (recorded/plex-members-run-2026-11-25.md, and each answer's shape in
	// recorded/response-shapes-2026-11-25.md). v1 routes answer XML
	// whatever Accept asks, their errors <Response code status>.
	sections: {api, "GET", "/api/servers/{machine}", formatXML},
	// Measured 2026-11-25: a JSON array, [] when none, with invited{id,
	// uuid} per share. owned/pending is the only listing with a pending
	// share's id.
	ownedAccepted: {api, "GET", "/api/v2/shared_servers/owned/accepted", formatJSON},
	ownedPending:  {api, "GET", "/api/v2/shared_servers/owned/pending", formatJSON},
	// Measured 2026-11-25 on plex.tv: 200 <SharedServer id>, whose id is
	// the share's. A second share to one account is 400 "You're already
	// sharing this server with {user}".
	share: {web, "POST", "/api/servers/{machine}/shared_servers", formatXML},
	// Measured 2026-11-25: 200 empty. After accept plex.tv answered and
	// clients.plex.tv wasn't tried; before accept clients.plex.tv answered
	// and plex.tv wasn't tried.
	deleteShare: {web, "DELETE", "/api/servers/{machine}/shared_servers/{share}", formatXML},
	// Measured 2026-11-25: <Invite id> is the invitee's account id, with
	// friend, home and server flags and a <Server> per pending share. A
	// share deleted before accept leaves its Invite with server="0" and no
	// <Server>.
	invites: {api, "GET", "/api/invites/requested", formatXML},
	// Measured 2026-11-25 on plex.tv: 200 <Response status="Canceled
	// invitation to ...">, with friend, home and server echoing the
	// Invite's.
	cancelInvite: {web, "DELETE", "/api/invites/requested/{invite}", formatXML},
	// Measured 2026-11-25: <User id uuid> per Home user.
	homeUsers: {api, "GET", "/api/home/users", formatXML},
	// Measured 2026-11-25: 403 "Invite a user requires a Plex Pass"
	// without Plex Pass, and 403 "You have reached the 15 person Home
	// limit" at 15 people. [unverified] past those: the run's Admin had no
	// Plex Pass.
	inviteHome: {web, "POST", "/api/home/users", formatXML},
	// [unverified]: the RUNBOOK's form, never called on a Home user.
	leaveHome: {web, "DELETE", "/api/home/users/{member}", formatXML},
	// The Member's own token, measured 2026-11-25 on clients.plex.tv: a
	// JSON array of invites, each with sharedServers[].id the share's id,
	// and accepting one by that id answers 204.
	pendingReceived: {api, "GET", "/api/v2/shared_servers/invites/received/pending", formatJSON},
	accept:          {api, "POST", "/api/v2/shared_servers/{share}/accept", formatJSON},
}
