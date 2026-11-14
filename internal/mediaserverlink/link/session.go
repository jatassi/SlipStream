package link

import "time"

// Session is one playback a server lists, as the kind read it.
type Session struct {
	// Key is the server's own session key.
	Key      string
	Path     ServerPath
	User     string
	Player   string
	Product  string
	Decision Decision
	// Hardware is whether the server reports hardware transcoding.
	Hardware bool
	// Reasons are the transcode reasons as the server gave them. Plex gives
	// none.
	Reasons []string
}

// Seen is one look at a server's sessions: a poll, or a push from its
// socket.
type Seen struct {
	At time.Time
	// Polled is whether this is a poll, which counts toward a session
	// missing from two polls. A push lists only what it carried.
	Polled   bool
	Sessions []Session
	// Transcoding is whether the server re-encodes any video now, counted
	// before the session cap and including a transcode no session lists.
	// A poll sets it.
	Transcoding bool
	// Items are item keys a push said changed, which make their index
	// entries stale.
	Items []string
	// Problem is set when the poll failed. Sessions is then empty and
	// means nothing.
	Problem *Problem
}
