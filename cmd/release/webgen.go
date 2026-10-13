package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
)

var webGeneratedPaths = []string{"web/src/api/gen", "web/src/routeTree.gen.ts"}

// webGenerated reports each generated web file the build just changed or
// added. Orval has no diff mode like sqlc's, so it reads git status.
func webGenerated(ctx context.Context, root string, w io.Writer) error {
	out, err := gitOutput(ctx, root, nil, append([]string{"status", "--porcelain", "-z", "--untracked-files=all", "--"}, webGeneratedPaths...)...)
	if err != nil {
		return err
	}
	var stale []string
	entries := bytes.Split(bytes.TrimRight(out, "\x00"), []byte{0})
	for i := 0; i < len(entries); i++ {
		entry := string(entries[i])
		if entry == "" {
			continue
		}
		// A rename or copy is followed by its source path.
		if entry[0] == 'R' || entry[0] == 'C' {
			i++
		}
		file := entry[3:]
		_, _ = fmt.Fprintf(w, "%s: generated code is stale or uncommitted; run bun run gen in web and commit it\n", file)
		stale = append(stale, file)
	}
	if len(stale) > 0 {
		return fmt.Errorf("stale: %s", strings.Join(stale, ", "))
	}
	return nil
}
