package world

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"github.com/jatassi/slipstream/fakes/internal/mediafix"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// EncodeFixtures fills the per-machine fixture cache with every catalog
// Scenario's release payloads and library files, using the tools beside
// binary as a launch does. A launch on a cold cache encodes them inside its
// deadline, beside every other world starting at once.
func EncodeFixtures(ctx context.Context, binary string) error {
	// The library recipes run their tools from a work folder.
	binary, err := filepath.Abs(binary)
	if err != nil {
		return fmt.Errorf("find the binary: %w", err)
	}
	cache, err := releases.CacheDir()
	if err != nil {
		return err
	}
	tools := mediafix.ToolsIn(filepath.Dir(binary), filepath.Ext(binary))
	b, err := mediafix.NewBuilder(ctx, tools, cache)
	if err != nil {
		return err
	}
	scratch, err := os.MkdirTemp("", "fakeworld-fixtures-")
	if err != nil {
		return fmt.Errorf("create a scratch folder: %w", err)
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	var files []scenario.MediaFile
	for i, name := range scenario.Catalog() {
		s, err := scenario.Load(name)
		if err != nil {
			return err
		}
		cfg := releases.Config{FFmpeg: tools.FFmpeg, Cache: cache, Dir: filepath.Join(scratch, strconv.Itoa(i)), Tracker: netip.MustParseAddrPort("127.0.0.1:1")}
		if _, err := releases.Build(ctx, s, cfg); err != nil {
			return fmt.Errorf("scenario %s: %w", name, err)
		}
		if s.Library != nil {
			for _, f := range s.Library.Folders {
				files = append(files, f.Files...)
			}
		}
	}
	return buildAll(ctx, b, files)
}

// buildAll builds files as many at a time as there are CPUs.
func buildAll(ctx context.Context, b *mediafix.Builder, files []scenario.MediaFile) error {
	slots := make(chan struct{}, runtime.NumCPU())
	errs := make([]error, len(files))
	var wg sync.WaitGroup
	for i, f := range files {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			_, errs[i] = b.Build(ctx, f)
		})
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
