package main

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/tokens"
)

func init() {
	register("gen", "write a committed generated source: tokens (the web theme from DESIGN.md)", runGen)
}

var generators = map[string]func(root string) error{
	"tokens": tokens.Write,
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
	return generate(root)
}
