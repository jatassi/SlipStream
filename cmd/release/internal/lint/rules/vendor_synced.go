package rules

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "vendor-synced",
		Doc:   "every file under web/vendor/<name>/ is a byte-identical copy of .claude/skills/<name>/; scripts/vendor-anatomy writes both (Nodes, Pairing and Roles spec #140, Vendor the anatomy skill)",
		Check: checkVendorSynced,
	})
}

const resync = "; never edit a vendored copy, re-run scripts/vendor-anatomy"

func checkVendorSynced(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := repo.Files()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		rel, ok := strings.CutPrefix(f, "web/vendor/")
		if !ok {
			continue
		}
		skill := path.Join(".claude/skills", rel)
		vendored, err := repo.Read(f)
		if err != nil {
			return nil, err
		}
		upstream, err := os.ReadFile(filepath.Join(repo.Root, filepath.FromSlash(skill)))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			findings = append(findings, lint.Finding{Path: f, Line: 1, Message: "has no counterpart at " + skill + resync})
		case err != nil:
			return nil, fmt.Errorf("read %s: %w", skill, err)
		case !bytes.Equal(vendored, upstream):
			findings = append(findings, lint.Finding{Path: f, Line: 1, Message: "differs from " + skill + resync})
		}
	}
	return findings, nil
}
