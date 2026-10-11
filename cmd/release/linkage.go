package main

import (
	"errors"

	"github.com/jatassi/slipstream/cmd/release/internal/linkage"
)

func init() {
	register("linkage", "check that binaries import only OS-provided libraries (P-352)", runLinkage)
}

func runLinkage(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: go run ./cmd/release linkage <binary>...")
	}
	var errs []error
	for _, path := range args {
		errs = append(errs, linkage.Check(path))
	}
	return errors.Join(errs...)
}
