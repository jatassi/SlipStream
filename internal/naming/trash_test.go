package naming

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// trashImport is what S53 (TRaSH Guides import) does to a TRaSH file
// template: it maps Radarr's {Edition Tags} to {Edition}, adds {Edition}
// where a template has no Edition token, and inserts " - {Target}".
func trashImport(file string) string {
	file = strings.ReplaceAll(file, "Edition Tags", "Edition")
	if !strings.Contains(file, "Edition}") {
		return file + " - {Edition} {Target}"
	}
	return file + " - {Target}"
}

// trashFolder is the folder rewrite that keeps a missing year from
// printing "()".
func trashFolder(folder string) string {
	return strings.ReplaceAll(folder, "({Release Year})", "{(Release Year)}")
}

// ungrouped are TRaSH's file templates that name a file without its Work
// or Item first, which the check refuses whatever the import does.
var ungrouped = map[string]bool{"original": true, "p2p-scene": true}

func readTrash(t *testing.T, file string, into any) {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, into); err != nil {
		t.Fatal(err)
	}
}

func TestTrashFilmTemplatesPassOnlyWithTheFolderRewrite(t *testing.T) {
	var trash struct{ Folder, File map[string]string }
	readTrash(t, "testdata/trash/radarr/naming/radarr-naming.json", &trash)
	checked := 0
	for key, file := range trash.File {
		folderKey := strings.NewReplacer("-anime", "", "-edition-alt", "").Replace(key)
		if key == "standard" || ungrouped[key] {
			folderKey = "default"
		}
		folder, ok := trash.Folder[folderKey]
		if !ok {
			t.Fatalf("file template %q has no folder %q", key, folderKey)
		}
		imported := Settings{Templates: templates("folder", trashFolder(folder), "file.standard", trashImport(file))}
		got := problemsOf(t, filmDecl, imported)
		if ungrouped[key] {
			if len(got) == 0 {
				t.Errorf("%s: %q passes; want it refused", key, file)
			}
			continue
		}
		checked++
		if got != nil {
			t.Errorf("%s: problems = %q; want none", key, got)
		}
		imported.Templates["folder"] = &folder
		got = problemsOf(t, filmDecl, imported)
		if len(got) != 1 || got[0].Key != "file.standard" || !strings.HasPrefix(got[0].Sentence, `For the sample "Nothing known" with no release facts, `) {
			t.Errorf("%s without the folder rewrite: problems = %q; want one on the sample with no year", key, got)
		}
	}
	if checked < 10 {
		t.Errorf("checked %d TRaSH film templates; the snapshot holds more", checked)
	}
}

func TestTrashEpisodeTemplatesPass(t *testing.T) {
	var trash struct {
		Season   map[string]string
		Shows    map[string]string `json:"series"`
		Episodes map[string]map[string]string
	}
	readTrash(t, "testdata/trash/sonarr/naming/sonarr-naming.json", &trash)
	variants := map[string]string{"standard": "file.standard", "daily": "file.daily", "anime": "file.absolute"}
	checked := 0
	for name, folder := range trash.Shows {
		imported := Settings{Templates: templates("folder", folder, "levels.1", trash.Season["default"])}
		for kind, field := range variants {
			imported.Templates[field] = ptr(trashImport(trash.Episodes[kind]["default"]))
		}
		checked++
		if got := problemsOf(t, showDecl, imported); got != nil {
			t.Errorf("%s: problems = %q; want none", name, got)
		}
	}
	if checked < 5 {
		t.Errorf("checked %d TRaSH folders; the snapshot holds more", checked)
	}
	for kind, files := range trash.Episodes {
		for key, file := range files {
			if !ungrouped[key] {
				continue
			}
			field := variants[kind]
			if got := problemsOf(t, showDecl, Settings{Templates: templates(field, trashImport(file))}); len(got) == 0 {
				t.Errorf("%s %s: %q passes; want it refused", kind, key, file)
			}
		}
	}
}
