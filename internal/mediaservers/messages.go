package mediaservers

import (
	"embed"
	"io/fs"

	"github.com/jatassi/slipstream/internal/mail"
)

//go:embed templates/*.tmpl
var templateFiles embed.FS

var templates = func() fs.FS {
	sub, err := fs.Sub(templateFiles, "templates")
	if err != nil {
		panic(err)
	}
	return sub
}()

var (
	plexAccountToAdd = mail.MustTemplate[toAddData](templates, "plex-account-to-add")
	canWatchOnPlex   = mail.MustTemplate[canWatchData](templates, "can-watch-on-plex")
)

type toAddData struct {
	Member, PlexUsername, PlexEmail, Server string
	AccountsURL                             string
}

type canWatchData struct {
	Server, PlexUsername string
	// Accepted is an active row. A shared row asks the Member to accept
	// Plex's email.
	Accepted bool
	// PlexPassLine is set while the Admin's Plex account has no Plex Pass.
	PlexPassLine bool
}
