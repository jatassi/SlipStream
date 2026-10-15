package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jatassi/slipstream/cmd/release/internal/pack"
	"github.com/jatassi/slipstream/cmd/release/internal/smoke"
)

func init() {
	register("smoke", "install each package release package wrote into -from, check it from outside and uninstall it; needs -disposable, since it changes this machine", runSmoke)
}

func runSmoke(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("smoke", flag.ContinueOnError)
	from := flags.String("from", "", "release package's output directory (required)")
	disposable := flags.Bool("disposable", false, "confirm this machine is disposable: the smoke installs packages system-wide and deletes their earlier data")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if *from == "" {
		return errors.New("-from is required")
	}
	if !*disposable {
		return errors.New("the smoke installs packages system-wide and deletes their earlier data dirs; run it only on a disposable machine, with -disposable")
	}
	m, err := pack.ReadManifest(*from)
	if err != nil {
		return err
	}
	if len(m.Packages) == 0 {
		return fmt.Errorf("%s lists no packages", *from)
	}
	return smoke.Smoke(ctx, &smoke.Run{Out: *from, Manifest: m, Log: os.Stderr})
}
