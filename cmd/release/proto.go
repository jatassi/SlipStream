package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const (
	protoDir      = "proto"
	pbDir         = "internal/pb"
	protoTemplate = "proto/buf.gen.yaml"
	frozenDir     = "slipstream/link/v1"
)

func init() {
	register("proto", "regenerate internal/pb from proto/ with the buf and generators tools/go.mod pins", runProto)
	register("proto-frozen", "fail when a change breaks slipstream.link.v1 against the latest v1+ release tag", runProtoFrozen)
}

func runProto(ctx context.Context, args []string) error {
	if err := flag.NewFlagSet("proto", flag.ContinueOnError).Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	// buf never deletes, so a removed .proto would leave its output behind.
	if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(pbDir))); err != nil {
		return fmt.Errorf("remove %s: %w", pbDir, err)
	}
	return bufGenerate(ctx, root, root)
}

// bufGenerate writes internal/pb under out. Its template runs
// cmd/protoc-gen-slipstream with go run, so it runs from the repo root.
func bufGenerate(ctx context.Context, root, out string) error {
	return run(ctx, root, nil, "go", "tool", "-modfile=tools/go.mod", "buf", "generate", "--template", protoTemplate, "--output", out)
}

// hostBuf resolves the buf tools/go.mod pins to its built binary, so it can
// run in a directory outside the module.
func hostBuf(ctx context.Context, root string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "tool", "-modfile=tools/go.mod", "-n", "buf")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go tool -n buf: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// protoCheck is release lint's proto step.
func protoCheck(ctx context.Context, root string, w io.Writer) error {
	return errors.Join(
		run(ctx, root, nil, "go", "tool", "-modfile=tools/go.mod", "buf", "lint", protoDir),
		frozenImports(root),
		protoDrift(ctx, root, filepath.Join(root, filepath.FromSlash(pbDir)), w),
	)
}

var protoImport = regexp.MustCompile(`^\s*import\s+(?:public\s+|weak\s+)?"([^"]+)"`)

// frozenImports keeps the frozen package free of anything a later version
// could change under it.
func frozenImports(root string) error {
	dir := filepath.Join(root, protoDir, filepath.FromSlash(frozenDir))
	files, err := filepath.Glob(filepath.Join(dir, "*.proto"))
	if err != nil {
		return fmt.Errorf("list %s: %w", frozenDir, err)
	}
	var errs []error
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read frozen proto: %w", err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			m := protoImport.FindStringSubmatch(line)
			if len(m) == 2 && !strings.HasPrefix(m[1], "google/protobuf/") {
				errs = append(errs, fmt.Errorf("%s/%s/%s:%d: imports %s; slipstream.link.v1 imports nothing outside google/protobuf", protoDir, frozenDir, filepath.Base(f), i+1, m[1]))
			}
		}
	}
	return errors.Join(errs...)
}

// protoDrift regenerates into a temporary directory and compares it with
// pb, since buf has no diff mode.
func protoDrift(ctx context.Context, root, pb string, w io.Writer) error {
	tmp, err := os.MkdirTemp("", "slipstream-proto-")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	if err := bufGenerate(ctx, root, tmp); err != nil {
		return err
	}
	want, err := readTree(filepath.Join(tmp, filepath.FromSlash(pbDir)))
	if err != nil {
		return err
	}
	got, err := readTree(pb)
	if err != nil {
		return err
	}
	var stale []string
	for name, b := range want {
		if g, ok := got[name]; !ok || !bytes.Equal(g, b) {
			stale = append(stale, name)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			stale = append(stale, name)
		}
	}
	if len(stale) == 0 {
		return nil
	}
	slices.Sort(stale)
	_, _ = fmt.Fprintf(w, "internal/pb is stale; run go run ./cmd/release proto\n")
	for _, name := range stale {
		_, _ = fmt.Fprintf(w, "  %s/%s\n", pbDir, name)
	}
	return errors.New("internal/pb is stale")
}

func readTree(dir string) (map[string][]byte, error) {
	files := map[string][]byte{}
	err := fs.WalkDir(os.DirFS(dir), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			return fmt.Errorf("read generated file: %w", err)
		}
		files[name] = b
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	return files, nil
}

func runProtoFrozen(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("proto-frozen", flag.ContinueOnError)
	base := flags.String("base", "", "revision the change merges into, such as origin/main, or HEAD^1 on main")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if *base == "" {
		return errors.New("-base is required")
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	return protoFrozen(ctx, os.Stdout, root, root, *base)
}

// protoFrozen runs buf breaking with the FILE rules, not WIRE, since Connect
// routes by /package.Service/Method and a rename breaks old Workers with
// unchanged bytes.
func protoFrozen(ctx context.Context, w io.Writer, root, dir, base string) error {
	tag, err := latestReleaseTag(ctx, dir, base)
	if err != nil {
		return err
	}
	if tag == "" {
		_, _ = fmt.Fprintln(w, "no v1+ release tag; slipstream.link.v1 is not frozen yet")
		return nil
	}
	buf, err := hostBuf(ctx, root)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, buf, "breaking", "--against", "../.git#tag="+tag+",subdir="+protoDir, "--path", frozenDir)
	cmd.Dir = filepath.Join(dir, protoDir)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("slipstream.link.v1 breaks against %s: %w", tag, err)
	}
	return nil
}

var releaseTag = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

// latestReleaseTag is the highest vX.Y.Z with X at least 1 merged into base,
// or "" when there is none.
func latestReleaseTag(ctx context.Context, dir, base string) (string, error) {
	out, err := gitOutput(ctx, dir, nil, "tag", "--merged", base, "--list", "v*")
	if err != nil {
		return "", err
	}
	var best string
	var bestVersion [3]int
	for tag := range strings.FieldsSeq(string(out)) {
		m := releaseTag.FindStringSubmatch(tag)
		if m == nil {
			continue
		}
		var v [3]int
		for i := range v {
			v[i], _ = strconv.Atoi(m[i+1])
		}
		if v[0] >= 1 && slices.Compare(v[:], bestVersion[:]) > 0 {
			best, bestVersion = tag, v
		}
	}
	return best, nil
}
