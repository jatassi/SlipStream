package main

import (
	"fmt"
	"maps"
	"path"
	"slices"
)

const noticesDir = "internal/notices"

// noticeFiles writes internal/notices/<key>/ for each upstream SlipStream
// ships code from: the upstream LICENSE verbatim, and a VENDORED.md whose
// Commit line the release notices read.
func noticeFiles(t *tree, cos map[string]checkout) error {
	for _, key := range slices.Sorted(maps.Keys(cos)) {
		co := cos[key]
		if co.Notice == "" {
			continue
		}
		dir := path.Join(noticesDir, key)
		t.own(dir)
		license, err := co.read(co.License)
		if err != nil {
			return err
		}
		t.add(path.Join(dir, "LICENSE"), license)
		t.add(path.Join(dir, "VENDORED.md"), vendored(co, co.Notice))
	}
	return nil
}

func vendored(co checkout, about string) []byte {
	return fmt.Appendf(nil, "# Ported: %s\n\nSource: %s\nCommit: %s\nLicense: %s (`LICENSE` beside this file)\n\n%s\n\n`make arr-sync` writes this file. To change it, edit cmd/arr-sync/pins.json and run it again.\n",
		co.Name, repoURL(co.Repo), co.Commit, co.SPDX, about)
}
