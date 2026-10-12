package main

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/cmd/release/internal/notices"
)

func init() {
	register("notices", "generate or check the third-party notices file and the license allow-list", runNotices)
}

func runNotices(ctx context.Context, args []string) error {
	if len(args) != 1 || (args[0] != "generate" && args[0] != "check") {
		return errors.New("usage: notices {generate|check}")
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	if err := buildWeb(ctx, root); err != nil {
		return err
	}
	if args[0] == "generate" {
		return notices.Write(ctx, root)
	}
	return notices.Check(ctx, root)
}
