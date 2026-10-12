package rules

import (
	"cmp"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

var (
	migrationName = regexp.MustCompile(`^(\d{5})_[a-z0-9]+(?:_[a-z0-9]+)*\.sql$`)
	gooseDown     = regexp.MustCompile(`(?i)^--\s*\+goose\s+down\b`)
)

func init() {
	lint.Register(lint.Rule{
		Name:  "migration-versions",
		Doc:   "every owner's schema/*.sql files are NNNNN_slug.sql, numbered 1, 2, 3 with no gap or repeat, and forward-only (spec #128, Checks)",
		Check: checkMigrationVersions,
	})
}

type migration struct {
	path    string
	version int
}

func checkMigrationVersions(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := repo.Files()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	sets := map[string][]migration{}
	for _, f := range files {
		if !lint.OwnerSQL(f, "schema") {
			continue
		}
		version := 0
		if m := migrationName.FindStringSubmatch(path.Base(f)); m != nil {
			version, _ = strconv.Atoi(m[1])
		}
		if version == 0 {
			findings = append(findings, lint.Finding{Path: f, Line: 1, Message: "name must be NNNNN_slug.sql"})
		} else {
			sets[path.Dir(f)] = append(sets[path.Dir(f)], migration{f, version})
		}
		src, err := repo.Read(f)
		if err != nil {
			return nil, err
		}
		for i, line := range strings.Split(string(src), "\n") {
			if gooseDown.MatchString(strings.TrimSpace(line)) {
				findings = append(findings, lint.Finding{Path: f, Line: i + 1, Message: "-- +goose Down found; migrations are forward-only (ADR 0015)"})
			}
		}
	}
	for _, set := range sets {
		findings = append(findings, versionFindings(set)...)
	}
	return findings, nil
}

func versionFindings(set []migration) []lint.Finding {
	slices.SortFunc(set, func(a, b migration) int {
		return cmp.Or(cmp.Compare(a.version, b.version), cmp.Compare(a.path, b.path))
	})
	var findings []lint.Finding
	previous := migration{}
	for _, m := range set {
		switch {
		case m.version == previous.version:
			findings = append(findings, lint.Finding{Path: m.path, Line: 1, Message: fmt.Sprintf("version %d is also %s", m.version, path.Base(previous.path))})
			continue
		case m.version > previous.version+1:
			findings = append(findings, lint.Finding{Path: m.path, Line: 1, Message: fmt.Sprintf("version %d follows %d; version %d is missing", m.version, previous.version, previous.version+1)})
		}
		previous = m
	}
	return findings
}
