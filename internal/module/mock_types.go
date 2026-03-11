package module

import (
	"context"
	"database/sql"

	"github.com/rs/zerolog"
)

// MockContext provides services and state to MockFactory methods.
type MockContext struct {
	DB     *sql.DB
	Logger *zerolog.Logger

	// Services available to mock factories
	RootFolderCreator MockRootFolderCreator
	QualityProfiles   []MockQualityProfile
	DefaultProfileID  int64
}

// MockRootFolderCreator creates root folders during dev mode setup.
type MockRootFolderCreator interface {
	Create(ctx context.Context, path, name, mediaType string) (rootFolderID int64, err error)
}

// MockQualityProfile is a minimal quality profile representation for mock factories.
type MockQualityProfile struct {
	ID   int64
	Name string
}
