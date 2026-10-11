package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

const (
	inputsPath = "native/inputs.json"
	recipeDir  = "native/recipe"
)

var depsCommands = map[string]func(native.Inputs, native.Target, string) error{
	"build": depsBuild,
	"env":   depsEnv,
}

func init() {
	register("deps", "build the native libs layer or print its cgo environment", runDeps)
}

func runDeps(args []string) error {
	names := make([]string, 0, len(depsCommands))
	for name := range depsCommands {
		names = append(names, name)
	}
	sort.Strings(names)
	usage := "usage: deps {" + strings.Join(names, "|") + "} [-target name]"
	if len(args) == 0 {
		return fmt.Errorf("%s", usage)
	}
	run, ok := depsCommands[args[0]]
	if !ok {
		return fmt.Errorf("unknown deps command %q; %s", args[0], usage)
	}
	fs := flag.NewFlagSet("deps "+args[0], flag.ContinueOnError)
	targetName := fs.String("target", native.HostTargetName(), "target from "+inputsPath)
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	in, err := native.LoadInputs(inputsPath)
	if err != nil {
		return err
	}
	target, err := in.Target(*targetName)
	if err != nil {
		return err
	}
	cacheRoot, err := native.CacheRoot()
	if err != nil {
		return err
	}
	return run(in, target, cacheRoot)
}

func depsBuild(in native.Inputs, t native.Target, cacheRoot string) error {
	if t.GOOS != runtime.GOOS || t.GOARCH != runtime.GOARCH {
		return fmt.Errorf("target %s builds on its own OS and architecture, not on %s", t.Name, native.HostTargetName())
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	dir, err := native.BuildLibs(ctx, in, t, recipeDir, cacheRoot, os.Stderr)
	if err != nil {
		return err
	}
	fmt.Println(dir)
	return nil
}

func depsEnv(in native.Inputs, t native.Target, cacheRoot string) error {
	tc, err := t.Compilers()
	if err != nil {
		return err
	}
	dir, err := native.FindLibs(in, t, recipeDir, cacheRoot)
	if err != nil {
		return err
	}
	md, err := native.ReadMetadata(dir)
	if err != nil {
		return err
	}
	fmt.Print(native.ExportLines(native.CgoEnv(dir, md, tc)))
	return nil
}
