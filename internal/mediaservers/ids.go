package mediaservers

import "github.com/jatassi/slipstream/internal/mediaservers/internal/db"

type (
	// AccessID is a Member's access to one Media Server.
	AccessID = db.AccessID
	// ProblemCode is the last failed call on an access row, for Admins only.
	ProblemCode = db.ProblemCode
)

// The problem codes an access row keeps, as the schema's CHECK lists them.
const (
	ProblemPlexRefused          ProblemCode = "plex_refused"
	ProblemPlexLibrariesShort   ProblemCode = "plex_libraries_short"
	ProblemUsernameTaken        ProblemCode = "username_taken"
	ProblemServerRefused        ProblemCode = "server_refused"
	ProblemMediaServerUnreached ProblemCode = "media_server_unreached"
	ProblemFailed               ProblemCode = "failed"
)
