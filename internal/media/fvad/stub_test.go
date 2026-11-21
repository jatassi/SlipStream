//go:build !cgo

package fvad_test

import (
	"errors"
	"testing"

	"github.com/jatassi/slipstream/internal/media/fvad"
	"github.com/jatassi/slipstream/internal/media/speech"
)

func TestNoCgoStub(t *testing.T) {
	if _, err := fvad.New(fvad.Mode, 16000); !errors.Is(err, fvad.ErrNoVAD) {
		t.Fatalf("New = %v, want ErrNoVAD", err)
	}
	if _, err := (fvad.Mapper{}).Map(t.Context(), "a.mkv", -1); !errors.Is(err, speech.ErrNodeFault) {
		t.Fatalf("Map = %v, want ErrNodeFault", err)
	}
}
