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
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/worktree"
)

func init() {
	register("tag", "check a release tag vX.Y.Z and write its version, pre-release flag and image tags; -notes writes its release notes; -backup adds a Backup the tag's released binary writes to "+releasesDir, runTag)
}

const imageRepo = "ghcr.io/jatassi/slipstream"

const installDoc = "https://github.com/jatassi/SlipStream/blob/%s/docs/install.md"

func runTag(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("tag", flag.ContinueOnError)
	notes := flags.String("notes", "", "also write the release notes to this file")
	backup := flags.String("backup", "", "the tag's released slipstream binary: take a Backup with it and add it to "+releasesDir)
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
	if *backup != "" {
		return addReleaseBackup(ctx, *backup, tag)
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

// addReleaseBackup adds the Backup binary writes, refusing a binary that
// isn't a clean build of tag.
func addReleaseBackup(ctx context.Context, binary, tag string) error {
	stamp, err := buildinfo.ReadFile(binary)
	if err != nil {
		return err
	}
	if stamp.Version != tag || stamp.Fingerprint != worktree.CleanFingerprint {
		return fmt.Errorf("%s is a build stamped %s with fingerprint %s; a release Backup takes a clean build of %s", binary, stamp.Version, stamp.Fingerprint, tag)
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	into, err := writeReleaseBackup(ctx, root, binary)
	if err != nil {
		return err
	}
	fmt.Println(into)
	return nil
}

func tagOutputs(v pack.Version) [][2]string {
	refs := make([]string, 0, 4)
	for _, t := range v.ImageTags() {
		refs = append(refs, imageRepo+":"+t)
	}
	return [][2]string{
		{"image", imageRepo},
		{"version", v.String()},
		{"prerelease", strconv.FormatBool(v.Prerelease())},
		{"image-tags", strings.Join(refs, " ")},
	}
}

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
