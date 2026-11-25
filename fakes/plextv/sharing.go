package plextv

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/plexaccount"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

var (
	// RefuseNextShare has plex.tv say no to the next share with a 400.
	RefuseNextShare = fake.CueSpec{Name: "plextv-refuse-next-share"}
	// RefuseNextAccept has plex.tv say no to a Member's next accept with a
	// 400.
	RefuseNextAccept = fake.CueSpec{Name: "plextv-refuse-next-accept"}
	// SignOutMember ends every token of {"account": "<username>"}, as the
	// Member signing SlipStream out on plex.tv.
	SignOutMember = fake.CueSpec{Name: "plextv-sign-out-member"}
	// HomeFull fills the Admin's Plex Home, so a Home invite gets the 403
	// for 15 people.
	HomeFull = fake.CueSpec{Name: "plextv-home-full"}
)

// Invites is the capture of each email plex.tv sends a Member when the
// Admin shares a server or invites them into the Plex Home, {"to":
// "<email>", "server": "<name>"} or {"to": "<email>", "home": true}, in
// plextv.invite records.
var Invites = fake.CaptureSpec{Name: "plex-invites", Kind: "plextv.invite"}

// Web is plex.tv's own host, which serves the share, its removal, the
// invite's cancel and the Plex Home changes, as the run of 2026-11-25
// called them there.
var Web = fake.Spec{
	Name: "plextv-web",
	TLS:  fake.FrontTLS,
	New:  newWeb,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.PlexTV.Web = endpoint.MustParseURL(at.URL(""))
	},
	Online: fake.NoOnlineRun("plex.tv sharing needs signed-in accounts; the sharing flow is a recorded manual run"),
}

// homeLimit is a Plex Home's size, the Admin included, in the run of
// 2026-11-25.
const homeLimit = 15

// share is one server shared with one account. Its id is also the id of
// the pending invite the account accepts.
type share struct {
	id          int64
	machine     string
	member      *person
	sections    []string
	inviteToken string
	invitedAt   time.Time
	acceptedAt  time.Time
	accepted    bool
}

// invite is what invites/requested lists for one account until it is
// accepted or canceled. It stays after a pending share is deleted, as the
// run of 2026-11-25 saw.
type invite struct {
	member  *person
	created time.Time
}

// sharing is the Admin's shares, invites and Plex Home, which s.mu
// guards.
type sharing struct {
	shares  []*share
	invites []*invite
	// home is the Plex Home's people besides the Admin. A Home invite adds
	// its account at once, [unverified] past the Plex Pass 403.
	home     []*person
	homeFull bool
	refuse   struct{ share, accept bool }
	mails    *fake.Capture
}

func newWeb(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
	s := stateOf(env.Shared)
	return fake.HTTP(env, fake.Routes{
		"share":         {Pattern: "POST /api/servers/{machine}/shared_servers", Handler: s.asAdmin(s.share)},
		"delete-share":  {Pattern: "DELETE /api/servers/{machine}/shared_servers/{share}", Handler: s.asAdmin(s.deleteShare)},
		"cancel-invite": {Pattern: "DELETE /api/invites/requested/{invite}", Handler: s.asAdmin(s.cancelInvite)},
		"invite-home":   {Pattern: "POST /api/home/users", Handler: s.asAdmin(s.inviteHome)},
		"leave-home":    {Pattern: "DELETE /api/home/users/{member}", Handler: s.asAdmin(s.leaveHome)},
	}), nil
}

func (s *state) sharingRoutes() fake.Routes {
	return fake.Routes{
		"sections":         {Pattern: "GET /api/servers/{machine}", Handler: s.asAdmin(s.sections)},
		"owned-accepted":   {Pattern: "GET /api/v2/shared_servers/owned/accepted", Handler: s.asAdmin(s.ownedShares(true))},
		"owned-pending":    {Pattern: "GET /api/v2/shared_servers/owned/pending", Handler: s.asAdmin(s.ownedShares(false))},
		"invites":          {Pattern: "GET /api/invites/requested", Handler: s.asAdmin(s.invitesRequested)},
		"home-users":       {Pattern: "GET /api/home/users", Handler: s.asAdmin(s.homeUsers)},
		"pending-received": {Pattern: "GET /api/v2/shared_servers/invites/received/pending", Handler: s.asMember(s.pendingReceived)},
		"accept":           {Pattern: "POST /api/v2/shared_servers/{share}/accept", Handler: s.asMember(s.accept)},
	}
}

func (s *state) sharingCues(env fake.Env) error {
	mails, err := env.Capture(Invites.Name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.sharing = sharing{mails: mails}
	s.mu.Unlock()
	return errors.Join(
		fake.OnCue(env, RefuseNextShare.Name, func(context.Context, struct{}) (any, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.refuse.share = true
			return nil, nil
		}),
		fake.OnCue(env, RefuseNextAccept.Name, func(context.Context, struct{}) (any, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.refuse.accept = true
			return nil, nil
		}),
		fake.OnCue(env, HomeFull.Name, func(context.Context, struct{}) (any, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.homeFull = true
			return nil, nil
		}),
		fake.OnCue(env, SignOutMember.Name, func(_ context.Context, a struct {
			Account string `json:"account"`
		}) (any, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			p := s.named(a.Account)
			if p == nil || p == s.admin() {
				return nil, fmt.Errorf("plextv-sign-out-member: plex.tv holds no Member account %q", a.Account)
			}
			for token, is := range s.issued {
				if is.device.account == p {
					delete(s.issued, token)
				}
			}
			return nil, nil
		}),
	)
}

// named is the account with username or email, nil for none. s.mu is
// held.
func (s *state) named(name string) *person {
	i := slices.IndexFunc(s.accounts, func(p *person) bool { return p.Username == name || p.Email == name })
	if i < 0 {
		return nil
	}
	return s.accounts[i]
}

// asAdmin serves h, with s.mu held, to the Admin's token, and 401 to any
// other.
func (s *state) asAdmin(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, d, ok := s.signedIn(r); !ok || d.account != s.admin() {
			unauthenticated(w)
			return
		}
		h(w, r)
	}
}

// asMember serves h, with s.mu held, to a signed-in account's token, and
// 401 to any other.
func (s *state) asMember(h func(http.ResponseWriter, *http.Request, *person)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, d, ok := s.signedIn(r)
		if !ok {
			unauthenticated(w)
			return
		}
		h(w, r, d.account)
	}
}

func writeXML(w http.ResponseWriter, v any) {
	writeXMLStatus(w, http.StatusOK, v)
}

func writeXMLStatus(w http.ResponseWriter, status int, v any) {
	b, err := xml.Marshal(v)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(append([]byte(xml.Header), b...))
}

// writeResponse is a v1 route's <Response code status>, which carries
// its errors and an invite's cancel, as the run of 2026-11-25 met it.
func writeResponse(w http.ResponseWriter, status int, message string) {
	writeXMLStatus(w, status, struct {
		XMLName xml.Name `xml:"Response"`
		Code    int      `xml:"code,attr"`
		Status  string   `xml:"status,attr"`
	}{Code: status, Status: message})
}

// sectionID is plex.tv's id of a server's section, which a share names
// and the server's own key isn't.
func sectionID(machine, key string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(machine + "/" + key))
	return strconv.FormatUint(uint64(10_000_000+h.Sum32()%90_000_000), 10)
}

type xmlSection struct {
	ID    string `xml:"id,attr"`
	Key   string `xml:"key,attr"`
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
}

// sections is the server's sections with their plex.tv ids, as PLEXAPI
// reads them and #74 met them.
func (s *state) sections(w http.ResponseWriter, r *http.Request) {
	srv, ok := s.owned.Server(r.PathValue("machine"))
	if !ok {
		writeResponse(w, http.StatusNotFound, "Server not found")
		return
	}
	type server struct {
		Name              string       `xml:"name,attr"`
		MachineIdentifier string       `xml:"machineIdentifier,attr"`
		Sections          []xmlSection `xml:"Section"`
	}
	out := server{Name: srv.Name, MachineIdentifier: srv.Identifier}
	for _, sec := range srv.Sections {
		out.Sections = append(out.Sections, xmlSection{ID: sectionID(srv.Identifier, sec.Key), Key: sec.Key, Title: sec.Title, Type: sec.Type})
	}
	writeXML(w, struct {
		XMLName xml.Name `xml:"MediaContainer"`
		Server  server   `xml:"Server"`
	}{Server: out})
}

const avatar = "https://plex.invalid/users/fake/avatar"

// myPlex and myPlexMachine are plex.tv's own identifier and machine
// identifier, which its v1 containers carry.
const (
	myPlex        = "com.plexapp.plugins.myplex"
	myPlexMachine = "fake-plex-tv"
)

// wire is an account as the shared_servers listings name it, its email
// only once the share is accepted.
func (p *person) wire(accepted bool) map[string]any {
	out := map[string]any{
		"id": p.id, "uuid": p.uuid, "username": p.Username, "title": p.Username, "thumb": avatar,
		"friendlyName": nil, "restricted": false, "subscription": nil,
	}
	if accepted {
		out["email"] = p.Email
	}
	return out
}

func sharingSettings() map[string]any {
	return map[string]any{
		"allowCameraUpload": false, "allowChannels": false, "allowSubtitleAdmin": false, "allowSync": false, "allowTuners": 0,
		"filterAll": nil, "filterMovies": nil, "filterMusic": nil, "filterPhotos": nil, "filterTelevision": nil,
	}
}

func timeOrNil(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

// serverID is plex.tv's number for a server, which a share carries beside
// its machine identifier.
func serverID(machine string) int64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(machine))
	return 1_000_000 + int64(h.Sum32()%9_000_000)
}

// wire is the share as shared_servers lists it, without invited and
// owner, as the run of 2026-11-25 met it.
func (s *state) wire(sh *share, owned bool) map[string]any {
	srv, _ := s.owned.Server(sh.machine)
	libraries := []map[string]any{}
	for _, sec := range srv.Sections {
		if id := sectionID(srv.Identifier, sec.Key); slices.Contains(sh.sections, id) {
			libraries = append(libraries, map[string]any{"id": json.Number(id), "key": json.Number(sec.Key), "title": sec.Title, "type": sec.Type})
		}
	}
	out := map[string]any{
		"id": sh.id, "accepted": sh.accepted, "acceptedAt": timeOrNil(sh.acceptedAt), "allLibraries": false, "deletedAt": nil,
		"invitedEmail": nil, "invitedId": sh.member.id, "lastSeenAt": sh.invitedAt.UTC().Format(time.RFC3339), "leftAt": nil,
		"machineIdentifier": sh.machine, "name": srv.Name, "numLibraries": len(libraries), "owned": owned, "ownerId": s.admin().id,
		"searchEnabled": true, "serverId": serverID(sh.machine), "libraries": libraries, "sharingSettings": sharingSettings(),
	}
	if owned {
		out["inviteToken"] = sh.inviteToken
	}
	return out
}

// ownedShares lists the Admin's accepted shares, or the pending ones,
// each with the share's id and the invited account, as the run of
// 2026-11-25 met them.
func (s *state) ownedShares(accepted bool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		out := []map[string]any{}
		for _, sh := range s.shares {
			if sh.accepted != accepted {
				continue
			}
			invited := sh.member.wire(accepted)
			invited["home"], invited["status"], invited["sharingSettings"] = slices.Contains(s.home, sh.member), "pending", sharingSettings()
			if accepted {
				invited["status"] = "accepted"
			}
			e := s.wire(sh, true)
			e["invited"], e["owner"] = invited, s.admin().wire(accepted)
			out = append(out, e)
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// share shares sections of machine with invited_id. A second share of one
// server to one account is the run's q5 400.
func (s *state) share(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SharedServer struct {
			Sections  []json.Number `json:"library_section_ids"`
			InvitedID int64         `json:"invited_id"`
		} `json:"shared_server"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeResponse(w, http.StatusBadRequest, "The request body isn't a share")
		return
	}
	srv, ok := s.owned.Server(r.PathValue("machine"))
	if !ok {
		writeResponse(w, http.StatusNotFound, "Server not found")
		return
	}
	i := slices.IndexFunc(s.accounts, func(p *person) bool { return p.id == body.SharedServer.InvitedID })
	if i < 0 || s.accounts[i] == s.admin() {
		writeResponse(w, http.StatusBadRequest, "User not found")
		return
	}
	member := s.accounts[i]
	if s.refuse.share {
		s.refuse.share = false
		writeResponse(w, http.StatusBadRequest, "The server couldn't be shared with this user")
		return
	}
	if slices.ContainsFunc(s.shares, func(sh *share) bool { return sh.machine == srv.Identifier && sh.member == member }) {
		writeResponse(w, http.StatusBadRequest, "You're already sharing this server with "+member.Username+". Please edit your existing share.")
		return
	}
	sections, ok := sectionsOf(srv, body.SharedServer.Sections)
	if !ok {
		writeResponse(w, http.StatusBadRequest, "A library section isn't the server's")
		return
	}
	now := time.Now()
	sh := &share{
		id: 100_000_000 + s.draw.Int64N(900_000_000), machine: srv.Identifier, member: member, sections: sections,
		inviteToken: s.word(32), invitedAt: now,
	}
	s.shares = append(s.shares, sh)
	if s.inviteOf(member) == nil {
		s.invites = append(s.invites, &invite{member: member, created: now})
	}
	s.mails.Record(r.Context(), map[string]any{"to": member.Email, "server": srv.Name})
	writeXML(w, sharedServerOf(srv, sh))
}

type xmlSharedSection struct {
	xmlSection
	Shared int `xml:"shared,attr"`
}

type xmlSharedServer struct {
	ID                 int64              `xml:"id,attr"`
	UserID             int64              `xml:"userID,attr"`
	Username           string             `xml:"username,attr"`
	Email              string             `xml:"email,attr"`
	Name               string             `xml:"name,attr"`
	Owned              int                `xml:"owned,attr"`
	InvitedAt          int64              `xml:"invitedAt,attr"`
	AcceptedAt         int64              `xml:"acceptedAt,attr"`
	AccessToken        string             `xml:"accessToken,attr"`
	AllowSync          int                `xml:"allowSync,attr"`
	AllowCameraUpload  int                `xml:"allowCameraUpload,attr"`
	AllowChannels      int                `xml:"allowChannels,attr"`
	AllowTuners        int                `xml:"allowTuners,attr"`
	AllowSubtitleAdmin int                `xml:"allowSubtitleAdmin,attr"`
	Sections           []xmlSharedSection `xml:"Section"`
}

// sharedServerOf is the share POST's answer, as the run of 2026-11-25
// met it.
func sharedServerOf(srv plexaccount.Server, sh *share) any {
	out := xmlSharedServer{
		ID: sh.id, UserID: sh.member.id, Username: sh.member.Username, Email: sh.member.Email, Name: srv.Name, InvitedAt: sh.invitedAt.Unix(),
	}
	for _, sec := range srv.Sections {
		if id := sectionID(srv.Identifier, sec.Key); slices.Contains(sh.sections, id) {
			out.Sections = append(out.Sections, xmlSharedSection{xmlSection{ID: id, Key: sec.Key, Title: sec.Title, Type: sec.Type}, 1})
		}
	}
	return struct {
		XMLName           xml.Name        `xml:"MediaContainer"`
		FriendlyName      string          `xml:"friendlyName,attr"`
		Identifier        string          `xml:"identifier,attr"`
		MachineIdentifier string          `xml:"machineIdentifier,attr"`
		Size              int             `xml:"size,attr"`
		SharedServer      xmlSharedServer `xml:"SharedServer"`
	}{FriendlyName: "myPlex", Identifier: myPlex, MachineIdentifier: myPlexMachine, Size: 1, SharedServer: out}
}

// sectionsOf is ids when each is a plex.tv id of srv's sections.
func sectionsOf(srv plexaccount.Server, ids []json.Number) ([]string, bool) {
	out := make([]string, len(ids))
	for i, id := range ids {
		if !slices.ContainsFunc(srv.Sections, func(sec plexaccount.Section) bool { return sectionID(srv.Identifier, sec.Key) == id.String() }) {
			return nil, false
		}
		out[i] = id.String()
	}
	return out, len(out) > 0
}

func (s *state) inviteOf(member *person) *invite {
	i := slices.IndexFunc(s.invites, func(inv *invite) bool { return inv.member == member })
	if i < 0 {
		return nil
	}
	return s.invites[i]
}

// deleteShare ends that one share and keeps its invite listed, as the
// run's q7 saw before accept.
func (s *state) deleteShare(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("share"), 10, 64)
	n := len(s.shares)
	s.shares = slices.DeleteFunc(s.shares, func(sh *share) bool { return sh.id == id && sh.machine == r.PathValue("machine") })
	if len(s.shares) == n {
		writeResponse(w, http.StatusNotFound, "Shared server not found")
		return
	}
	w.WriteHeader(http.StatusOK)
}

type xmlInviteServer struct {
	Name         string `xml:"name,attr"`
	NumLibraries int    `xml:"numLibraries,attr"`
}

type xmlInvite struct {
	ID           int64             `xml:"id,attr"`
	Username     string            `xml:"username,attr"`
	Email        string            `xml:"email,attr"`
	FriendlyName string            `xml:"friendlyName,attr"`
	Thumb        string            `xml:"thumb,attr"`
	CreatedAt    int64             `xml:"createdAt,attr"`
	Friend       int               `xml:"friend,attr"`
	Home         int               `xml:"home,attr"`
	Server       int               `xml:"server,attr"`
	Servers      []xmlInviteServer `xml:"Server"`
}

// invitesRequested lists each invite by the account's id, with a Server
// per pending share and no share id, as the run of 2026-11-25 met it.
// An invite whose pending shares were deleted is server="0" with none.
func (s *state) invitesRequested(w http.ResponseWriter, _ *http.Request) {
	out := []xmlInvite{}
	for _, inv := range s.invites {
		x := xmlInvite{
			ID: inv.member.id, Username: inv.member.Username, Email: inv.member.Email, Thumb: avatar,
			CreatedAt: inv.created.Unix(), Friend: 1,
		}
		for _, sh := range s.shares {
			if sh.member == inv.member && !sh.accepted {
				srv, _ := s.owned.Server(sh.machine)
				x.Servers = append(x.Servers, xmlInviteServer{Name: srv.Name, NumLibraries: len(sh.sections)})
			}
		}
		if len(x.Servers) > 0 {
			x.Server = 1
		}
		out = append(out, x)
	}
	writeXML(w, struct {
		XMLName xml.Name    `xml:"MediaContainer"`
		Invites []xmlInvite `xml:"Invite"`
	}{Invites: out})
}

// cancelInvite removes the account's invite and leaves its shares.
func (s *state) cancelInvite(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("invite"), 10, 64)
	i := slices.IndexFunc(s.invites, func(inv *invite) bool { return inv.member.id == id })
	if i < 0 {
		writeResponse(w, http.StatusNotFound, "Invite not found")
		return
	}
	member := s.invites[i].member
	s.invites = slices.Delete(s.invites, i, i+1)
	writeResponse(w, http.StatusOK, "Canceled invitation to "+member.Username)
}

type xmlUser struct {
	ID                 int64  `xml:"id,attr"`
	UUID               string `xml:"uuid,attr"`
	Username           string `xml:"username,attr"`
	Email              string `xml:"email,attr"`
	Title              string `xml:"title,attr"`
	Thumb              string `xml:"thumb,attr"`
	Admin              int    `xml:"admin,attr"`
	Guest              int    `xml:"guest,attr"`
	HasPassword        int    `xml:"hasPassword,attr"`
	Protected          int    `xml:"protected,attr"`
	Restricted         int    `xml:"restricted,attr"`
	RestrictionProfile string `xml:"restrictionProfile,attr"`
}

// homeUsers is the Plex Home, the Admin first, as the run of 2026-11-25
// met it less guestUserID and totalSize, which plex.tv sometimes leaves
// out.
func (s *state) homeUsers(w http.ResponseWriter, _ *http.Request) {
	var out []xmlUser
	for _, p := range append([]*person{s.admin()}, s.home...) {
		admin := 0
		if p == s.admin() {
			admin = 1
		}
		out = append(out, xmlUser{
			ID: p.id, UUID: p.uuid, Username: p.Username, Email: p.Email, Title: p.Username, Thumb: avatar,
			Admin: admin, HasPassword: 1, Protected: 1,
		})
	}
	writeXML(w, struct {
		XMLName           xml.Name  `xml:"MediaContainer"`
		FriendlyName      string    `xml:"friendlyName,attr"`
		Identifier        string    `xml:"identifier,attr"`
		MachineIdentifier string    `xml:"machineIdentifier,attr"`
		Size              int       `xml:"size,attr"`
		Users             []xmlUser `xml:"User"`
	}{FriendlyName: "myPlex", Identifier: myPlex, MachineIdentifier: myPlexMachine, Size: len(out), Users: out})
}

// inviteHome is the run's q6: 403 while the Admin has no Plex Pass, and
// 403 once the Home holds 15 people. Past them it adds the account to the
// Home at once, [unverified].
func (s *state) inviteHome(w http.ResponseWriter, r *http.Request) {
	switch {
	case !s.admin().PlexPass:
		writeResponse(w, http.StatusForbidden, "Invite a user requires a Plex Pass")
		return
	case s.homeFull || 1+len(s.home) >= homeLimit:
		writeResponse(w, http.StatusForbidden, "You have reached the 15 person Home limit")
		return
	}
	p := s.named(r.URL.Query().Get("invitedEmail"))
	if p == nil || p == s.admin() {
		writeResponse(w, http.StatusNotFound, "User not found")
		return
	}
	if !slices.Contains(s.home, p) {
		s.home = append(s.home, p)
		s.mails.Record(r.Context(), map[string]any{"to": p.Email, "home": true})
	}
	w.WriteHeader(http.StatusOK)
}

// leaveHome removes the account from the Home, [unverified].
func (s *state) leaveHome(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("member"), 10, 64)
	n := len(s.home)
	s.home = slices.DeleteFunc(s.home, func(p *person) bool { return p.id == id })
	if len(s.home) == n {
		writeResponse(w, http.StatusNotFound, "User not found")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// pendingReceived is the Member's invites from the Admin, each pending
// share's id in sharedServers, as the run of 2026-11-25 met them.
func (s *state) pendingReceived(w http.ResponseWriter, _ *http.Request, member *person) {
	out := []map[string]any{}
	servers := []map[string]any{}
	for _, sh := range s.shares {
		if sh.member == member && !sh.accepted {
			servers = append(servers, s.wire(sh, false))
		}
	}
	if len(servers) > 0 {
		out = append(out, map[string]any{
			"type": "received", "friendshipStatus": "pending", "invitedId": member.id, "ownerId": s.admin().id,
			"invited": member.wire(false), "owner": s.admin().wire(false), "sharedServers": servers,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// accept is the Member taking a pending share by its id, 204 as #74 and
// the run's q4 saw. The share is then accepted and off the invite.
func (s *state) accept(w http.ResponseWriter, r *http.Request, member *person) {
	id, _ := strconv.ParseInt(r.PathValue("share"), 10, 64)
	i := slices.IndexFunc(s.shares, func(sh *share) bool { return sh.id == id && sh.member == member && !sh.accepted })
	if i < 0 {
		writeError(w, http.StatusNotFound, 1020, "Invite not found")
		return
	}
	if s.refuse.accept {
		s.refuse.accept = false
		writeError(w, http.StatusBadRequest, 1999, "The invite couldn't be accepted")
		return
	}
	sh := s.shares[i]
	sh.accepted, sh.acceptedAt = true, time.Now()
	if !slices.ContainsFunc(s.shares, func(o *share) bool { return o.member == member && !o.accepted }) {
		s.invites = slices.DeleteFunc(s.invites, func(inv *invite) bool { return inv.member == member })
	}
	w.WriteHeader(http.StatusNoContent)
}
