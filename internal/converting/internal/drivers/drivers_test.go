package drivers_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/converting/internal/drivers"
)

func TestTheReaderAnswers(t *testing.T) {
	var ffmpeg string
	for _, p := range filepath.SplitList(os.Getenv("SLIPSTREAM_NATIVE_TOOLS")) {
		if name := filepath.Base(p); name == "ffmpeg" || name == "ffmpeg.exe" {
			ffmpeg = p
		}
	}
	if ffmpeg == "" {
		t.Fatal("SLIPSTREAM_NATIVE_TOOLS names no ffmpeg: run the tests with go run ./cmd/release test")
	}
	adapters, err := drivers.Local(ffmpeg, os.Environ())(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range adapters {
		if a.Driver == "" {
			t.Errorf("%+v has no driver version", a)
		}
	}
	t.Logf("adapters: %+v", adapters)
}
