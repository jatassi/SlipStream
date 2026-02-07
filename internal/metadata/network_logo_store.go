package metadata

import (
	"context"
	"database/sql"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// SQLNetworkLogoStore implements NetworkLogoStore using the database.
type SQLNetworkLogoStore struct {
	queries *sqlc.Queries
}

// NewSQLNetworkLogoStore creates a new SQLNetworkLogoStore.
func NewSQLNetworkLogoStore(db *sql.DB) *SQLNetworkLogoStore {
	return &SQLNetworkLogoStore{queries: sqlc.New(db)}
}

// SetDB updates the database connection.
func (s *SQLNetworkLogoStore) SetDB(db *sql.DB) {
	s.queries = sqlc.New(db)
}

func (s *SQLNetworkLogoStore) UpsertNetworkLogo(ctx context.Context, name, logoURL string) error {
	return s.queries.UpsertNetworkLogo(ctx, sqlc.UpsertNetworkLogoParams{
		Name:    name,
		LogoUrl: logoURL,
	})
}

func (s *SQLNetworkLogoStore) GetNetworkLogoURL(ctx context.Context, name string) (string, error) {
	logo, err := s.queries.GetNetworkLogo(ctx, name)
	if err != nil {
		return "", err
	}
	return logo.LogoUrl, nil
}

func (s *SQLNetworkLogoStore) GetAllNetworkLogos(ctx context.Context) (map[string]string, error) {
	logos, err := s.queries.GetAllNetworkLogos(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(logos))
	for _, logo := range logos {
		result[logo.Name] = logo.LogoUrl
	}
	return result, nil
}
