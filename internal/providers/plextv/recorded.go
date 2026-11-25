package plextv

// Facts are what the recorded Plex run settled that changes what SlipStream
// offers.
type Facts struct {
	// AdminShares is false when the Admin's JWT is refused on the share or
	// section routes. Share on Plex and Add to Plex Home then aren't
	// offered, and Share by hand and Mark as added still are.
	AdminShares bool
	// AdminInvitesHome is false when the Admin's JWT can't invite an
	// existing account into the Plex Home.
	AdminInvitesHome bool
	// HomeNeedsPlexPass hides Add to Plex Home while the Admin has no Plex
	// Pass.
	HomeNeedsPlexPass bool
}

// Recorded is the run of 2026-11-25. The Admin's JWT shared on plex.tv. A
// Plex Home invite needed Plex Pass (403 "Invite a user requires a Plex
// Pass"). AdminInvitesHome is true on the spec's word: the run couldn't
// pass the Plex Pass gate to see an invite succeed.
var Recorded = Facts{AdminShares: true, AdminInvitesHome: true, HomeNeedsPlexPass: true}
