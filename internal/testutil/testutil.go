// Package testutil provides testing utilities for integration tests.
package testutil

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database"
)

// TestDB wraps a test database connection.
type TestDB struct {
	DB      *database.DB
	Manager *database.Manager
	Conn    *sql.DB
	Path    string
	Logger  zerolog.Logger
}

// NewTestDB creates a new test database in a temp directory.
// It runs migrations and returns a ready-to-use database.
// The caller should defer Close() to clean up.
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "slipstream_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	devDBPath := filepath.Join(tmpDir, "test_dev.db")

	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).Level(zerolog.DebugLevel)

	// Create database manager
	manager, err := database.NewManager(dbPath, devDBPath, &logger)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create database manager: %v", err)
	}

	// Run migrations
	if err := manager.Migrate(); err != nil {
		manager.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return &TestDB{
		Manager: manager,
		Conn:    manager.Conn(),
		Path:    tmpDir,
		Logger:  logger,
	}
}

// Close closes the database and removes the temp directory.
func (tdb *TestDB) Close() {
	if tdb.Manager != nil {
		tdb.Manager.Close()
	} else if tdb.DB != nil {
		tdb.DB.Close()
	}
	if tdb.Path != "" {
		os.RemoveAll(tdb.Path)
	}
}

// NewTestLogger creates a test logger that outputs to t.Log.
func NewTestLogger(t *testing.T) zerolog.Logger {
	t.Helper()
	return zerolog.New(zerolog.NewTestWriter(t)).Level(zerolog.DebugLevel)
}

// NopLogger returns a no-op logger for tests that don't need output.
func NopLogger() zerolog.Logger {
	return zerolog.Nop()
}

// StringPtr returns a pointer to a string.
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to an int.
func IntPtr(i int) *int {
	return &i
}

// Int64Ptr returns a pointer to an int64.
func Int64Ptr(i int64) *int64 {
	return &i
}

// BoolPtr returns a pointer to a bool.
func BoolPtr(b bool) *bool {
	return &b
}
