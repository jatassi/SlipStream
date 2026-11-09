// Command capture writes the ffprobe captures internal/media's tests
// parse, from mediafix's recipes, with the bundled tools that
// SLIPSTREAM_NATIVE_TOOLS lists:
//
//	go run ./fakes/internal/mediafix/capture internal/media/testdata/ffprobe
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jatassi/slipstream/fakes/internal/mediafix"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: capture <dir>")
		os.Exit(2)
	}
	if err := capture(context.Background(), os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "capture:", err)
		os.Exit(1)
	}
}

func capture(ctx context.Context, dir string) error {
	tools, err := mediafix.NativeTools()
	if err != nil {
		return err
	}
	cache, err := os.MkdirTemp("", "mediafix-")
	if err != nil {
		return fmt.Errorf("make a fixture cache: %w", err)
	}
	defer func() { _ = os.RemoveAll(cache) }()
	b, err := mediafix.NewBuilder(ctx, tools, cache)
	if err != nil {
		return err
	}
	for _, c := range mediafix.Captures {
		if err := b.Capture(ctx, c, dir); err != nil {
			return err
		}
	}
	return nil
}
