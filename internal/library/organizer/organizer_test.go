package organizer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func newTestService() *Service {
	logger := zerolog.New(zerolog.NewTestWriter(nil)).Level(zerolog.Disabled)
	return NewService(&logger)
}

func TestNewService(t *testing.T) {
	logger := zerolog.Nop()

	service := NewService(&logger)

	if service == nil {
		t.Fatal("NewService() returned nil")
	}
}

func TestService_CopyFile(t *testing.T) {
	service := newTestService()

	tmpDir, err := os.MkdirTemp("", "organizer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(srcPath, []byte("test content"), 0o644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	destPath := filepath.Join(tmpDir, "subdir", "dest.txt")
	if err := service.CopyFile(srcPath, destPath); err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}

	if _, err := os.Stat(srcPath); err != nil {
		t.Error("CopyFile() removed source file")
	}

	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("CopyFile() dest file not found: %v", err)
	}

	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read dest file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("CopyFile() content = %q, want %q", string(content), "test content")
	}
}

func TestService_CopyFile_NonExistentSource(t *testing.T) {
	service := newTestService()

	tmpDir, err := os.MkdirTemp("", "organizer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "nonexistent.mkv")
	destPath := filepath.Join(tmpDir, "dest.mkv")

	err = service.CopyFile(srcPath, destPath)
	if err == nil {
		t.Error("CopyFile() with non-existent source should return error")
	}
}
