package api

import (
	"strings"
	"testing"
)

func TestAllSwitchableServicesRegistered(t *testing.T) {
	// setupTestServer calls NewServer which calls Validate() at startup.
	// If any switchable field is nil, NewServer would Fatal. This test
	// explicitly re-validates to make the assertion visible in test output.
	ts, cleanup := setupTestServer(t)
	defer cleanup()

	if err := ts.switchable.Validate(); err != nil {
		t.Fatalf("Switchable services validation failed: %v", err)
	}
}

func TestSwitchableServices_UpdateAll_NoPanic(t *testing.T) {
	// Verify UpdateAll completes without panic on a fully wired server.
	ts, cleanup := setupTestServer(t)
	defer cleanup()

	db := ts.dbManager.Conn()
	ts.switchable.UpdateAll(db)
}

func TestSwitchableServices_Validate_DetectsNilFields(t *testing.T) {
	sw := SwitchableServices{}
	err := sw.Validate()
	if err == nil {
		t.Fatal("Validate() should return error when all required fields are nil")
	}

	// Should mention at least some of the nil fields
	for _, name := range []string{"Defaults", "Movies", "TV", "Quality"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("Validate() error should mention %q, got: %s", name, err.Error())
		}
	}
}

func TestSwitchableServices_Validate_SkipsOptional(t *testing.T) {
	// Build a fully populated struct except for the optional Passkey field.
	ts, cleanup := setupTestServer(t)
	defer cleanup()

	// Nil out the optional field
	ts.switchable.Passkey = nil

	err := ts.switchable.Validate()
	if err != nil {
		t.Fatalf("Validate() should not fail when only optional fields are nil, got: %v", err)
	}
}
