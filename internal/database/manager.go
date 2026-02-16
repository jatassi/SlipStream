package database

import (
	"database/sql"
	"os"
	"sync"

	"github.com/rs/zerolog"
)

// Manager manages production and development database connections
// and provides runtime switching between them.
type Manager struct {
	prodDB    *DB
	devDB     *DB
	devMode   bool
	mu        sync.RWMutex
	devDBPath string
	logger    *zerolog.Logger
}

// NewManager creates a new database manager with the production database.
// The development database is lazy-loaded when first needed.
func NewManager(prodPath, devPath string, logger *zerolog.Logger) (*Manager, error) {
	prodDB, err := New(prodPath)
	if err != nil {
		return nil, err
	}

	return &Manager{
		prodDB:    prodDB,
		devDBPath: devPath,
		logger:    logger,
	}, nil
}

// Conn returns the currently active database connection.
// Returns the dev database connection when in dev mode, otherwise production.
func (m *Manager) Conn() *sql.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.devMode && m.devDB != nil {
		return m.devDB.Conn()
	}
	return m.prodDB.Conn()
}

// ProdConn returns the production database connection, regardless of dev mode.
// This is useful for copying data from production to dev database.
func (m *Manager) ProdConn() *sql.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.prodDB.Conn()
}

// SetDevMode switches between production and development databases.
// When enabling dev mode, any existing dev database is deleted and a fresh one is created.
// This ensures mock data is always freshly populated with the latest schema and data.
func (m *Manager) SetDevMode(enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if enabled {
		// Always start fresh: close existing dev DB and delete the file
		if m.devDB != nil {
			m.logger.Info().Msg("closing existing development database")
			m.devDB.Close()
			m.devDB = nil
		}

		// Delete the dev database file to ensure fresh start
		if err := os.Remove(m.devDBPath); err != nil && !os.IsNotExist(err) {
			m.logger.Warn().Err(err).Str("path", m.devDBPath).Msg("failed to delete dev database file")
		}

		m.logger.Info().Str("path", m.devDBPath).Msg("creating fresh development database")

		devDB, err := New(m.devDBPath)
		if err != nil {
			return err
		}

		if err := devDB.Migrate(); err != nil {
			devDB.Close()
			return err
		}

		m.devDB = devDB
	}

	m.devMode = enabled
	m.logger.Info().Bool("devMode", enabled).Msg("developer mode changed")

	return nil
}

// IsDevMode returns whether developer mode is currently active.
func (m *Manager) IsDevMode() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.devMode
}

// Migrate runs migrations on the production database.
func (m *Manager) Migrate() error {
	return m.prodDB.Migrate()
}

// Close closes both database connections.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var prodErr, devErr error

	if m.prodDB != nil {
		prodErr = m.prodDB.Close()
	}

	if m.devDB != nil {
		devErr = m.devDB.Close()
	}

	if prodErr != nil {
		return prodErr
	}
	return devErr
}
