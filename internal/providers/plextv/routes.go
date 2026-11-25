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
	// The Admin's JWT on the sharing routes, from the Plex run of
	// 2026-11-25 (recorded/plex-members-run-2026-11-25.md), whose q1 got
	// 200 on clients.plex.tv and plex.tv for every listing below. Bodies
	// and parameters are the run's RUNBOOK's.
	sections:      {api, "GET", "/api/servers/{machine}", formatXML},
	ownedAccepted: {api, "GET", "/api/v2/shared_servers/owned/accepted", formatJSON},
	// q2: the only listing that carries a pending share's id.
	ownedPending: {api, "GET", "/api/v2/shared_servers/owned/pending", formatJSON},
	// q1: 200 on plex.tv, the only host tried. q5: a second share to the
	// same account is 400 "You're already sharing this server".
	share: {web, "POST", "/api/servers/{machine}/shared_servers", formatJSON},
	// q7: 200 before and after accept. The report doesn't name the host
	// that answered; plex.tv is the RUNBOOK's.
	deleteShare: {web, "DELETE", "/api/servers/{machine}/shared_servers/{share}", formatJSON},
	// q2: lists the invite without the share id. q7: a share deleted
	// before accept leaves its invite here.
	invites: {api, "GET", "/api/invites/requested", formatXML},
	// q7: 200, with friend, server and home copied from the invite. The
	// report doesn't name the host that answered; plex.tv is the RUNBOOK's.
	cancelInvite: {web, "DELETE", "/api/invites/requested/{invite}", formatJSON},
	homeUsers:    {api, "GET", "/api/home/users", formatXML},
	// q6: 403 "Invite a user requires a Plex Pass" without Plex Pass, and
	// 403 "You have reached the 15 person Home limit" at 15 people.
	// [unverified] past those: the run's Admin had no Plex Pass.
	inviteHome: {web, "POST", "/api/home/users", formatJSON},
	// [unverified]: the RUNBOOK's form, never called on a Home user.
	leaveHome: {web, "DELETE", "/api/home/users/{member}", formatJSON},
	// The Member's own token. Measured by #74 and q4: the pending invite's
	// id is the share's, and accepting it answers 204.
	pendingReceived: {api, "GET", "/api/v2/shared_servers/invites/received/pending", formatJSON},
	accept:          {api, "POST", "/api/v2/shared_servers/{share}/accept", formatJSON},
}
