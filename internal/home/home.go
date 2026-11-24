// Package home owns Home's one setting, Rows on Home, and reads what Home
// shows of the Grabs, the Imports, the Moves and their Works in one
// snapshot.
package home

import (
	"context"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// SettingsChanged tells Admins that Rows on Home changed.
type SettingsChanged struct{}

var SettingsChangedEvent = live.AdminEvent[SettingsChanged]("home.settings-changed")

type Publisher[T any] interface {
	Publish(ctx context.Context, v T)
}

// Moves is the Moves that need the Admin: *moves.Moves.
type Moves interface {
	NeedsYou(ctx context.Context, r storage.Reader[file], limit int) ([]moves.Need, error)
}

type Config struct {
	Library *storage.DB[file]
	Moves   Moves
	Grabs   *grabs.Grabs
	Imports *imports.Imports
	Works   *works.Works
	Files   *files.Files
	Clients *downloadclients.Clients
	Changed Publisher[SettingsChanged]
}

type Service struct{ cfg Config }

func New(cfg Config) *Service { return &Service{cfg: cfg} }
