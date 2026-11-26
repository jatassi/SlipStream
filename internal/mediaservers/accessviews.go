package mediaservers

import (
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
)

// AccessFilter narrows the Admin's access list.
type AccessFilter struct {
	// Account narrows the list to one Member's rows and adds the servers
	// they may be offered. Zero lists every row.
	Account identity.AccountID
}

// AccessList is every access row an Admin reads, Needs you first.
type AccessList struct {
	Rows         []AccessView   `json:"rows" nullable:"false" doc:"The rows under Needs you first, then the rest"`
	Offerable    []AccessServer `json:"offerable,omitempty" doc:"With account: the servers shown to Members that the Member has no access to yet"`
	PlexPass     bool           `json:"plexPass" doc:"Whether SlipStream's Plex account has Plex Pass, which Add to Plex Home needs"`
	PlexHomeUsed bool           `json:"plexHomeUsed" doc:"Whether a Member is in the Plex Home through SlipStream: a shared or active row added to Plex Home"`
}

// AccessView is one Member's access to one Media Server, as an Admin reads
// it.
type AccessView struct {
	ID        string         `json:"id" format:"uuid"`
	Server    AccessServer   `json:"server"`
	Member    AccessMember   `json:"member"`
	State     access.State   `json:"state" enum:"offered,waiting,shared,making,active,removing,removed"`
	NeedsYou  bool           `json:"needsYou" doc:"Whether the row waits on an Admin: a share to make or accept, or a removal to try again"`
	Actions   []access.Event `json:"actions" nullable:"false" enum:"shareFriend,shareHome,markAdded,check,link,offer,retryRemove,remove" doc:"What this Admin may start on the row, in the order the screen shows them"`
	Plex      *PlexPart      `json:"plex,omitempty" doc:"Plex rows only"`
	User      *UserPart      `json:"user,omitempty" doc:"Jellyfin and Emby rows with a user on the server"`
	Libraries []string       `json:"libraries" nullable:"false" doc:"The server's libraries for Members by name, for sharing by hand"`
	Problem   *AccessProblem `json:"problem,omitempty" doc:"The last call that failed on the row; absent once a move succeeds"`
	ChangedAt time.Time      `json:"changedAt"`
}

// AccessServer is the server of an access row.
type AccessServer struct {
	ID     string         `json:"id" format:"uuid"`
	Name   string         `json:"name"`
	Kind   api.Enum[Kind] `json:"kind"`
	Access string         `json:"access" enum:"plex_tv,users" doc:"How Members get access: plex_tv through a Plex share, users with a user the Member makes or the Admin links"`
}

// AccessMember is the Member of an access row.
type AccessMember struct {
	AccountID string `json:"accountId,omitempty" format:"uuid" doc:"Absent once the Account is deleted and the row waits for its removal"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty" format:"email" doc:"Absent once the Account is deleted"`
	Disabled  bool   `json:"disabled" doc:"Whether the Account is turned off"`
}

// PlexPart is the Plex side of an access row.
type PlexPart struct {
	Username string `json:"username,omitempty" doc:"The Member's Plex username; absent until the Member signs in to Plex"`
	Email    string `json:"email,omitempty" format:"email" doc:"The Member's Plex email; absent until the Member signs in to Plex"`
	PlexHome bool   `json:"plexHome" doc:"True when the Member is offered or added to the Plex Home; false for a share as a friend"`
	ByHand   bool   `json:"byHand" doc:"The Admin shared the server in Plex and marked it as added"`
}

// UserPart is the server user of a Jellyfin or Emby access row.
type UserPart struct {
	Username string `json:"username"`
}

// AccessProblem is the last failed call on an access row, for Admins only.
type AccessProblem struct {
	Code   ProblemCode `json:"code" enum:"plex_refused,plex_libraries_short,username_taken,server_refused,media_server_unreached,failed"`
	Detail string      `json:"detail,omitempty" doc:"What went wrong, as a sentence for an Admin"`
}

// Acted is an Admin's operation on an access row.
type Acted struct {
	// Row is nil when the move deleted the row.
	Row *AccessView
	// Mail is set when the move sent "You can watch {server} on Plex".
	Mail *mail.Outcome
}

// MyAccess is a Member's own access, for the Watch step, How to watch and
// Where to watch.
type MyAccess struct {
	Rows         []MyAccessRow `json:"rows" nullable:"false" doc:"In the Admin's server order"`
	Plex         *MyPlex       `json:"plex,omitempty" doc:"The Plex account the Member signed in to Plex as; absent before a sign-in"`
	PlexPassLine bool          `json:"plexPassLine" doc:"Whether How to watch carries Plex's Plex Pass rule: the Member has a Plex row and SlipStream's Plex account has no Plex Pass"`
	MailSetUp    bool          `json:"mailSetUp" doc:"Whether SlipStream can send email, so the Member hears when a share is made"`
}

type MyPlex struct {
	Username string `json:"username"`
}

// MyAccessRow is one row of a Member's own access. It carries no problem:
// a Member reads a server's reply only in the answer to their own call.
type MyAccessRow struct {
	ID         string         `json:"id" format:"uuid"`
	Server     MyServer       `json:"server"`
	AdminName  string         `json:"adminName" doc:"The Admin to ask about this server"`
	State      access.State   `json:"state" enum:"offered,waiting,shared,making,active"`
	Pending    bool           `json:"pending" doc:"Whether the Member still has something to do or wait for"`
	Actions    []access.Event `json:"actions" nullable:"false" enum:"check,anotherPlex,makeAccount" doc:"What the Member may start on the row"`
	HowToWatch *HowToWatch    `json:"howToWatch,omitempty" doc:"Absent while pending"`
}

type MyServer struct {
	Name   string `json:"name"`
	Label  string `json:"label" doc:"The kind's product name, such as Jellyfin"`
	Access string `json:"access" enum:"plex_tv,users"`
}

// HowToWatch is what a Member needs to watch on a server in its app.
type HowToWatch struct {
	App      string `json:"app" doc:"The app to get: Plex, Jellyfin or Emby"`
	SignInAs string `json:"signInAs" doc:"The Plex username, or the username on the server"`
	Address  string `json:"address,omitempty" format:"uri" doc:"The server's Address for Members; absent on Plex, and when the Admin set none"`
}

// MemberPin is a Member's pending Plex sign-in.
type MemberPin struct {
	ID        int64     `json:"pinId"`
	AuthURL   string    `json:"authUrl" format:"uri" doc:"Open it in a new tab, or in this one on a phone, where the Member signs in to Plex"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// PinState is where a Member's Plex sign-in stands.
type PinState string

const (
	PinWaiting  PinState = "waiting"
	PinExpired  PinState = "expired"
	PinSignedIn PinState = "signed_in"
	PinRefused  PinState = "refused"
)

// PinRefusal is why a Member's Plex sign-in was refused.
type PinRefusal string

const (
	// PlexAccountTaken is a Plex account another Account signed in with.
	PlexAccountTaken PinRefusal = "plex_account_taken"
	// PlexAccountOwner is the Plex account SlipStream's Admin signed in
	// with, which owns the servers.
	PlexAccountOwner PinRefusal = "plex_account_owner"
	// PlexAccountOther is a different Plex account while a share names the
	// one the Member signed in with before.
	PlexAccountOther PinRefusal = "plex_account_other"
)

// MemberPinState is a Member's Plex sign-in as the page polls it.
type MemberPinState struct {
	State   PinState   `json:"state" enum:"waiting,expired,signed_in,refused"`
	Refusal PinRefusal `json:"refusal,omitempty" enum:"plex_account_taken,plex_account_owner,plex_account_other" doc:"Set when state is refused"`
}

// Invite is how an Admin shares a Plex server with a Member.
type Invite string

const (
	InviteFriend Invite = "friend"
	InviteHome   Invite = "home"
)

// ServerUser is a user on a Jellyfin or Emby server an Admin may link to a
// Member.
type ServerUser struct {
	ID   string `json:"id" doc:"The server's user id"`
	Name string `json:"name"`
}

// LibraryChoice is one of a server's Libraries for Members.
type LibraryChoice struct {
	Key  string `json:"key" minLength:"1" maxLength:"256" doc:"Plex's section key, or the Jellyfin or Emby library ItemId"`
	Name string `json:"name" minLength:"1" maxLength:"256"`
}
