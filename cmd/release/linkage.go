package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/linkage"
	"github.com/jatassi/slipstream/internal/install"
)

func init() {
	register("linkage", "check that binaries import only OS-provided libraries (P-352), and that each Windows binary has its role's subsystem", runLinkage)
}

func runLinkage(_ context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: go run ./cmd/release linkage <binary>")
	}
	var errs []error
	for _, path := range args {
		errs = append(errs, linkage.Check(path, roleOf(path)))
	}
	return errors.Join(errs...)
}

func roleOf(path string) linkage.Role {
	if strings.TrimSuffix(filepath.Base(path), ".exe") == install.Trampoline {
		return linkage.Trampoline
	}
	return linkage.Node
}
