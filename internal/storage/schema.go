package storage

import (
	"fmt"
	"io/fs"
	"maps"
	"regexp"
	"slices"
	"strconv"
)

// Schema is one owner's migration set in file F: a set name and the
// owner's schema/ folder. An owner makes it in its Schema function:
//
//	//go:embed schema/*.sql
//	var schemaFiles embed.FS
//
//	func Schema() storage.Schema[file] { return storage.NewSchema[file]("clip", schemaFiles) }
//
// Because the type carries F, OpenLibrary takes only []Schema[Library] and
// OpenLocal only []Schema[Local]; a schema cannot land in the wrong file.
type Schema[F File] struct {
	set      string
	files    fs.FS   // rooted at the owner's schema/ folder
	versions []int64 // every NNNNN in files, ascending
	latest   int64   // the highest version this Schema applies
}

var (
	setName       = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	migrationName = regexp.MustCompile(`^(\d{5})_[a-z0-9_]+\.sql$`)
)

// NewSchema reads owner's schema/*.sql. set names the goose_<set> version
// table and prefixes every table the set creates; it never changes.
//
// It panics when set does not match [a-z][a-z0-9]*, is "goose" or
// "sqlite", or when schema/ holds no NNNNN_slug.sql file: those are
// programmer errors, and the audit over the composition root's lists
// reaches every Schema in a test. Version gaps, duplicates and Down
// sections are release lint's migration-versions findings.
func NewSchema[F File](set string, owner fs.FS) Schema[F] {
	if !setName.MatchString(set) || set == "goose" || set == "sqlite" {
		panic(fmt.Sprintf("storage: schema set name %q must match [a-z][a-z0-9]* and not be goose or sqlite", set))
	}
	sub, err := fs.Sub(owner, "schema")
	if err != nil {
		panic(fmt.Sprintf("storage: schema %s: %v", set, err))
	}
	entries, err := fs.ReadDir(sub, ".")
	if err != nil {
		panic(fmt.Sprintf("storage: schema %s: %v", set, err))
	}
	var versions []int64
	for _, e := range entries {
		if m := migrationName.FindStringSubmatch(e.Name()); m != nil {
			v, _ := strconv.ParseInt(m[1], 10, 64)
			versions = append(versions, v)
		}
	}
	if len(versions) == 0 {
		panic("storage: schema " + set + " has no schema/NNNNN_slug.sql file")
	}
	slices.Sort(versions)
	return Schema[F]{set: set, files: sub, versions: versions, latest: versions[len(versions)-1]}
}

func (s Schema[F]) Set() string { return s.set }

// Through is the schema cut at version: what an older binary shipped.
// Upgrade tests open a file with the cut schema, then with the whole one.
func (s Schema[F]) Through(version int64) Schema[F] {
	if version < 1 || version > s.latest {
		panic(fmt.Sprintf("storage: schema %s has no version %d to cut at", s.set, version))
	}
	s.latest = version
	return s
}

// above is every version in files the Schema does not apply.
func (s Schema[F]) above() []int64 {
	i, _ := slices.BinarySearch(s.versions, s.latest+1)
	return s.versions[i:]
}

// Versions is a file's schema version vector: each set's highest applied
// migration. A set the file has never run is absent, not zero.
type Versions map[string]int64

// Plan is what opening a file applies: one Step per set with pending
// migrations, in schema-list order. An empty Plan applies nothing and never
// calls the pre-migration hook.
type Plan []Step

type Step struct {
	Set  string `json:"set"`
	From int64  `json:"from"`
	To   int64  `json:"to"`
}

// Compare checks the versions a file holds against the binary's schema list
// and plans the migrations opening it would apply. It is pure: startup runs
// it on the versions read from the file, and Backups and restore runs it on
// the versions a Backup holds, so restore refuses exactly what startup
// refuses.
//
// The error is a *SchemaError: a held version above the list's latest for
// that set, or a held set the list lacks. Sets are checked in list order,
// then unknown sets in name order, so the first finding is deterministic.
func Compare[F File](held Versions, schemas []Schema[F]) (Plan, error) {
	p := policyOf[F]()
	var plan Plan
	known := map[string]bool{}
	for _, s := range schemas {
		known[s.set] = true
		h := held[s.set]
		if h > s.latest {
			return nil, &SchemaError{File: p.name, Set: s.set, Held: h, Known: s.latest, advice: p.newerAdvice}
		}
		if h < s.latest {
			plan = append(plan, Step{Set: s.set, From: h, To: s.latest})
		}
	}
	for _, set := range slices.Sorted(maps.Keys(held)) {
		if !known[set] {
			return nil, &SchemaError{File: p.name, Set: set, Held: held[set]}
		}
	}
	return plan, nil
}

// SchemaError is a file whose schema this binary cannot run. Known is 0
// when the binary has no such set; every real Schema's latest is >= 1.
type SchemaError struct {
	File  string
	Set   string
	Held  int64
	Known int64

	advice string
}

func (e *SchemaError) Error() string {
	if e.Known == 0 {
		return fmt.Sprintf("%s holds schema %s, which this binary does not have", e.File, e.Set)
	}
	return fmt.Sprintf("%s holds schema %s at version %d, but this binary knows only up to %d%s", e.File, e.Set, e.Held, e.Known, e.advice)
}
