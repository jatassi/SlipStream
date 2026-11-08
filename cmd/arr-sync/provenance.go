package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// record is one owner's upstream.json, by upstream key: the commit, each
// hand-ported Go file mapped to the upstream files it ports (S27 (Ported
// site indexers)'s shape), and the sha256 of every upstream file the
// owner reads, so a resync can name what changed under each Go file.
type record map[string]*provenance

type provenance struct {
	Commit string              `json:"commit"`
	Date   string              `json:"date"`
	Files  map[string][]string `json:"files,omitempty"`
	SHA256 map[string]string   `json:"sha256"`
}

// provenanceFiles writes <owner>/upstream.json for every owner and
// returns the review list of each change since the record it replaces.
func provenanceFiles(t *tree, root *os.Root, p pins, cos map[string]checkout) ([]string, error) {
	reads := ownerReads(p)
	var review []string
	for _, owner := range slices.Sorted(maps.Keys(reads)) {
		rec := record{}
		for from, files := range reads[owner] {
			pr, err := ownerProvenance(cos[from], files, p.Ports[owner])
			if err != nil {
				return nil, err
			}
			rec[from] = pr
		}
		file := path.Join(owner, "upstream.json")
		old, err := readRecord(root, file)
		if err != nil {
			return nil, err
		}
		review = append(review, changes(owner, old, rec, cos)...)
		b, err := json.MarshalIndent(rec, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", file, err)
		}
		t.add(file, append(b, '\n'))
	}
	return review, nil
}

func ownerReads(p pins) map[string]map[string][]string {
	reads := map[string]map[string][]string{}
	read := func(owner, from, file string) {
		if reads[owner] == nil {
			reads[owner] = map[string][]string{}
		}
		reads[owner][from] = append(reads[owner][from], file)
	}
	for _, pf := range p.Patterns {
		read(pf.Owner, pf.From, pf.File)
		for _, o := range pf.Fields {
			read(o, pf.From, pf.File)
		}
	}
	for _, set := range p.Fixtures {
		for _, f := range set.Files {
			read(set.Owner, set.From, path.Join(set.Dir, f))
		}
	}
	formats := path.Dir(p.Dialects.Out)
	for _, e := range p.Dialects.Enums {
		read(formats, e.From, e.File)
	}
	for _, f := range slices.Concat(p.Dialects.Languages, p.Dialects.Iso) {
		read(formats, f.From, f.File)
	}
	for _, f := range p.Trash.Readers {
		read(p.Trash.ReadersOwner, p.Trash.From, f)
	}
	for owner, files := range p.Ports {
		for _, refs := range files {
			for _, ref := range refs {
				from, file, _ := strings.Cut(ref, ":")
				read(owner, from, file)
			}
		}
	}
	return reads
}

func ownerProvenance(co checkout, files []string, ports map[string][]string) (*provenance, error) {
	pr := &provenance{Commit: co.Commit, Date: co.date, SHA256: map[string]string{}}
	for _, f := range files {
		b, err := co.read(f)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(b)
		pr.SHA256[f] = hex.EncodeToString(sum[:])
	}
	for goFile, refs := range ports {
		for _, ref := range refs {
			if k, file, _ := strings.Cut(ref, ":"); k == co.key {
				if pr.Files == nil {
					pr.Files = map[string][]string{}
				}
				pr.Files[goFile] = append(pr.Files[goFile], file)
			}
		}
	}
	return pr, nil
}

func readRecord(root *os.Root, file string) (record, error) {
	b, err := root.ReadFile(filepath.FromSlash(file))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}
	var rec record
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return rec, nil
}

// changes lists, for one owner, each upstream file whose hash differs from
// the old record, with the Go files that port it: the review list.
func changes(owner string, old, rec record, cos map[string]checkout) []string {
	var out []string
	for _, from := range slices.Sorted(maps.Keys(rec)) {
		was, ok := old[from]
		if !ok || was.Commit == rec[from].Commit {
			continue
		}
		now := rec[from]
		for _, file := range slices.Sorted(maps.Keys(now.SHA256)) {
			if was.SHA256[file] == now.SHA256[file] {
				continue
			}
			var ports []string
			for goFile, srcs := range now.Files {
				if slices.Contains(srcs, file) {
					ports = append(ports, path.Join(owner, goFile))
				}
			}
			slices.Sort(ports)
			line := fmt.Sprintf("%s %s changed (%s..%s)", cos[from].Name, file, was.Commit[:8], now.Commit[:8])
			if len(ports) > 0 {
				line += ". Review " + strings.Join(ports, ", ")
			}
			out = append(out, line)
		}
	}
	return out
}
