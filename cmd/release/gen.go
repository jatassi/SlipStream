package main

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/icons"
	"github.com/jatassi/slipstream/cmd/release/internal/tokens"
)

func init() {
	register("gen", "write a committed generated source: tokens (the web theme from DESIGN.md) or icons (the web app's PNG icons from brand/app-icon.svg)", runGen)
}

var generators = map[string]func(ctx context.Context, root string) error{
	"tokens": func(_ context.Context, root string) error { return tokens.Write(root) },
	"icons":  genIcons,
}

func genIcons(ctx context.Context, root string) error {
	if err := installWeb(ctx, root); err != nil {
		return err
	}
	if err := run(ctx, root, nil, "bun", icons.Renderer); err != nil {
		return err
	}
	return icons.Record(root)
}

func runGen(ctx context.Context, args []string) error {
	names := strings.Join(slices.Sorted(maps.Keys(generators)), "|")
	if len(args) != 1 {
		return fmt.Errorf("usage: gen {%s}", names)
	}
	generate, ok := generators[args[0]]
	if !ok {
		return fmt.Errorf("unknown generator %q; usage: gen {%s}", args[0], names)
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	return generate(ctx, root)
}
