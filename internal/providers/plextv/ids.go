package plextv

import "log/slog"

// AccountID is plex.tv's id of a Plex account.
type AccountID int64

// SectionID is plex.tv's id of one library section of a server, as a
// share names it.
type SectionID string

// ShareID is plex.tv's id of one share, which is also the id of the
// pending invite the Member accepts.
type ShareID string

// Who is a Plex account as an access row keeps it, so a removal works once
// the Account is gone.
type Who struct {
	ID   AccountID
	UUID string
}

// Invitee is who a Plex Home invite addresses.
type Invitee struct {
	Who
	Username, Email string
}

// MemberToken is a Member's legacy Plex token. String, GoString and
// LogValue print [token].
type MemberToken struct{ s string }

func NewMemberToken(s string) MemberToken { return MemberToken{s: s} }

func (t MemberToken) Reveal() string { return t.s }

func (MemberToken) String() string { return "[token]" }

func (MemberToken) GoString() string { return "[token]" }

func (MemberToken) LogValue() slog.Value { return slog.StringValue("[token]") }
