package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/cmd/release/internal/pack"
	"github.com/jatassi/slipstream/internal/install"
)

func init() {
	register("tag", "check a release tag vX.Y.Z and write its version, pre-release flag and image tags; -notes writes its release notes", runTag)
}

// imageRepo is the registry repository every Linux image is pushed to.
const imageRepo = "ghcr.io/jatassi/slipstream"

// installDoc is the install guide the release notes link to, at the tag.
const installDoc = "https://github.com/jatassi/SlipStream/blob/%s/docs/install.md"

func runTag(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("tag", flag.ContinueOnError)
	notes := flags.String("notes", "", "also write the release notes to this file")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if flags.NArg() != 1 {
		return errors.New("usage: release tag [-notes file] vX.Y.Z")
	}
	tag := flags.Arg(0)
	v, err := pack.ParseTag(tag)
	if err != nil {
		return err
	}
	for _, o := range tagOutputs(v) {
		fmt.Printf("%s=%s\n", o[0], o[1])
		if err := githubOutput(o[0], o[1]); err != nil {
			return err
		}
	}
	if *notes == "" {
		return nil
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	in, err := native.LoadInputs(filepath.Join(root, native.InputsPath))
	if err != nil {
		return err
	}
	if err := os.WriteFile(*notes, []byte(releaseNotes(tag, v, in)), 0o644); err != nil {
		return fmt.Errorf("write the release notes: %w", err)
	}
	return nil
}

// tagOutputs are the step outputs the release jobs read, in order.
func tagOutputs(v pack.Version) [][2]string {
	refs := make([]string, 0, 4)
	for _, t := range v.ImageTags() {
		refs = append(refs, imageRepo+":"+t)
	}
	return [][2]string{
		{"version", v.String()},
		{"prerelease", strconv.FormatBool(v.Prerelease())},
		{"image-tags", strings.Join(refs, " ")},
	}
}

// releaseNotes states the oldest system each package runs on and links the
// install guide, which walks through each system's first-launch prompts.
func releaseNotes(tag string, v pack.Version, in native.Inputs) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n\n## Supported systems\n\n", install.ProductName, v)
	for _, goos := range []string{"windows", "darwin"} {
		for _, t := range in.Targets {
			if t.GOOS == goos {
				fmt.Fprintf(&b, "- %s or later (%s)\n", minOS(t.MinOS), t.Name)
			}
		}
	}
	fmt.Fprintf(&b, "- Linux with glibc %s and kernel %s or later (%s)\n", install.GlibcFloor, install.KernelFloor, strings.Join(linuxTargets(in), ", "))
	fmt.Fprintf(&b, "\nInstall steps for each system: "+installDoc+"\n", tag)
	return b.String()
}

// minOS spells a target's min_os, such as "macos 13.0", as a person reads it.
func minOS(s string) string {
	name, version, _ := strings.Cut(s, " ")
	switch name {
	case "windows":
		name = "Windows"
	case "macos":
		name = "macOS"
	}
	return name + " " + version
}

func linuxTargets(in native.Inputs) []string {
	var names []string
	for _, t := range in.Targets {
		if t.GOOS == "linux" {
			names = append(names, t.Name)
		}
	}
	return names
}
