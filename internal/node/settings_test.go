package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node/internal/config"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

func defaultView(t *testing.T, kind install.Kind) view {
	t.Helper()
	cfg, keys, err := loadConfig(fstest.MapFS{}, nil, noTypes, kind)
	if err != nil {
		t.Fatal(err)
	}
	return view{saved: &cfg, running: &cfg, defaults: &cfg, keys: keysByName(keys), dir: DataDir{path: testDataPath}, self: testNodeID}
}

var (
	testDataPath = filepath.Join(string(filepath.Separator)+"srv", "slipstream")
	testNodeID   = role.NodeID{UUID: storage.NewUUID()}
)

func jsonText(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestTheNodeSettingsTableReadsItsDefaultsFromDefaults(t *testing.T) {
	for _, kind := range install.Kinds() {
		cfg := Defaults(noTypes, kind)
		leaves := map[string]bool{}
		for _, leaf := range config.Walk(&cfg) {
			leaves[leaf.Path] = true
		}
		for _, r := range nodeSettings {
			if !leaves[r.key()] {
				t.Errorf("%s: row %s is not a config.yaml key", kind, r.key())
			}
		}
		cfg.CheckConfig(func(key string, err error) { t.Errorf("%s: default %s fails its check: %v", kind, key, err) })
	}

	got, err := json.Marshal(defaultView(t, install.Docker).fields())
	if err != nil {
		t.Fatal(err)
	}
	downloads := jsonText(t, filepath.Join(testDataPath, "downloads"))
	folder := jsonText(t, filepath.Join(testDataPath, "backups"))
	scratch := jsonText(t, filepath.Join(testDataPath, "converting"))
	want := `[` +
		`{"key":"http.listen","label":"Listen on","source":"default","applies":"restart","isDefault":true,"pendingRestart":false,"kind":"listen","value":{"host":"all","port":8484,"address":"[::]:8484"},"running":{"host":"all","port":8484,"address":"[::]:8484"},"default":{"host":"all","port":8484,"address":"[::]:8484"}},` +
		`{"key":"http.ports_to_try","label":"Ports to try","source":"default","applies":"restart","isDefault":true,"pendingRestart":false,"kind":"count","value":1,"running":1,"default":1,"min":1,"max":100,"unit":{"one":"port","many":"ports"}},` +
		`{"key":"log.level","label":"Level","source":"default","applies":"live","isDefault":true,"pendingRestart":false,"kind":"choice","value":"info","running":"info","default":"info","options":[{"value":"debug","label":"Debug"},{"value":"info","label":"Info"},{"value":"warn","label":"Warn"},{"value":"error","label":"Error"}]},` +
		`{"key":"log.max_size_mb","label":"Rotate at","source":"default","applies":"live","isDefault":true,"pendingRestart":false,"kind":"count","value":10,"running":10,"default":10,"min":1,"max":1000,"unit":{"one":"MB","many":"MB"}},` +
		`{"key":"log.max_files","label":"Files kept","source":"default","applies":"live","isDefault":true,"pendingRestart":false,"kind":"count","value":5,"running":5,"default":5,"min":0,"max":100,"unit":{"one":"file","many":"files"},"zero":"No limit"},` +
		`{"key":"log.max_age_days","label":"Days kept","source":"default","applies":"live","isDefault":true,"pendingRestart":false,"kind":"count","value":30,"running":30,"default":30,"min":0,"max":3650,"unit":{"one":"day","many":"days"},"zero":"No limit"},` +
		`{"key":"log.compress","label":"Compress rotated files","source":"default","applies":"live","isDefault":true,"pendingRestart":false,"kind":"switch","value":true,"running":true,"default":true},` +
		`{"key":"log.viewer_lines","label":"Lines kept for the viewer","source":"default","applies":"live","isDefault":true,"pendingRestart":false,"kind":"count","value":2000,"running":2000,"default":2000,"min":100,"max":50000,"unit":{"one":"line","many":"lines"}},` +
		`{"key":"torrent.listen","label":"Listen address","source":"default","applies":"restart","isDefault":true,"pendingRestart":false,"kind":"address","value":"0.0.0.0:6881","running":"0.0.0.0:6881","default":"0.0.0.0:6881"},` +
		`{"key":"torrent.downloads","label":"Downloads folder","source":"default","applies":"restart","isDefault":true,"pendingRestart":false,"kind":"folder","value":` + downloads + `,"running":` + downloads + `,"default":` + downloads + `,"nodeId":"` + testNodeID.String() + `","purpose":"downloads"},` +
		`{"key":"backups.folder","label":"Backups folder","source":"default","applies":"restart","isDefault":true,"pendingRestart":false,"kind":"folder","value":` + folder + `,"running":` + folder + `,"default":` + folder + `,"nodeId":"` + testNodeID.String() + `","purpose":"backups"},` +
		`{"key":"converting.scratch","label":"Scratch folder","source":"default","applies":"restart","isDefault":true,"pendingRestart":false,"kind":"folder","value":` + scratch + `,"running":` + scratch + `,"default":` + scratch + `,"nodeId":"` + testNodeID.String() + `","purpose":"converting"},` +
		`{"key":"converting.ffmpeg","label":"ffmpeg path","source":"default","applies":"restart","isDefault":true,"pendingRestart":false,"kind":"file","value":"","running":"","default":"","nodeId":"` + testNodeID.String() + `","unsupported":true}` +
		`]`
	if string(got) != want {
		t.Fatalf("fields =\n%s\nwant\n%s", got, want)
	}
}

func TestAFieldNamesItsLockAndAPendingRestart(t *testing.T) {
	v := defaultView(t, install.Dev)
	saved := *v.saved
	saved.HTTP.PortsToTry = 3
	v.saved = &saved
	v.keys["http.ports_to_try"] = Key{Key: "http.ports_to_try", Value: "3", Default: "10", Source: config.SourceDotEnv, Env: "SLIPSTREAM_HTTP_PORTS_TO_TRY"}
	got, err := json.Marshal(v.fields()[1])
	if err != nil {
		t.Fatal(err)
	}
	want := `{"key":"http.ports_to_try","label":"Ports to try","source":".env","applies":"restart","isDefault":false,"lockedBy":"SLIPSTREAM_HTTP_PORTS_TO_TRY","pendingRestart":true,"kind":"count","value":3,"running":10,"default":10,"min":1,"max":100,"unit":{"one":"port","many":"ports"}}`
	if string(got) != want {
		t.Fatalf("field = %s, want %s", got, want)
	}
}

func testConfigFile(t *testing.T, yaml, dotEnv string) configFile {
	t.Helper()
	dir := testDataDir(t)
	if yaml != "" {
		writeFile(t, filepath.Join(dir.Path(), "config.yaml"), yaml)
	}
	if dotEnv != "" {
		writeFile(t, filepath.Join(dir.Path(), ".env"), dotEnv)
	}
	return configFile{dir: dir, types: noTypes, kind: install.Dev}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readConfigYAML(t *testing.T, f configFile) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(f.dir.Path(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSetWritesAKeyAndUnsetReturnsItToItsDefault(t *testing.T) {
	f := testConfigFile(t, "# kept\nname: media\n", "")
	saved, keys, err := f.set(t.Context(), nil, []nodesystem.Change{{Key: "log.level", Value: nodesystem.Text("debug")}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := readConfigYAML(t, f), "# kept\nname: media\nlog:\n  level: debug\n"; got != want {
		t.Fatalf("config.yaml = %q, want %q", got, want)
	}
	if k := keysByName(keys)["log.level"]; saved.Log.Level != "debug" || k.Source != config.SourceYAML {
		t.Fatalf("log.level = %s from %s, want debug from config.yaml", saved.Log.Level, k.Source)
	}

	saved, keys, err = f.set(t.Context(), nil, []nodesystem.Change{{Key: "log.level", Value: nodesystem.Reset()}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := readConfigYAML(t, f), "# kept\nname: media\n"; got != want {
		t.Fatalf("config.yaml = %q, want %q", got, want)
	}
	if k := keysByName(keys)["log.level"]; saved.Log.Level != "info" || k.Source != config.SourceDefault {
		t.Fatalf("log.level = %s from %s, want info from default", saved.Log.Level, k.Source)
	}
}

func TestSetWithNoChangesLeavesConfigYAMLAsWritten(t *testing.T) {
	const yaml = "name: media\n\n# the log\nlog:\n  max_files: 3\n"
	f := testConfigFile(t, yaml, "")
	saved, _, err := f.set(t.Context(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := readConfigYAML(t, f); got != yaml || saved.Log.MaxFiles != 3 {
		t.Fatalf("config.yaml = %q with max_files %d, want it untouched with 3", got, saved.Log.MaxFiles)
	}
}

// refusedAs holds when err is a Refusal with want's key, reason and
// variable, whose sentence names want.Sentence, a fact such as a bound.
func refusedAs(err error, want nodesystem.Refusal) bool {
	refused, ok := errors.AsType[*nodesystem.Refusal](err)
	if !ok || refused.Sentence == "" || !strings.Contains(refused.Sentence, want.Sentence) {
		return false
	}
	got := *refused
	got.Sentence = want.Sentence
	return got == want
}

func TestSetRefusesAndChangesNothing(t *testing.T) {
	const yaml = "log:\n  max_files: 3\n"
	for _, c := range []struct {
		name    string
		dotEnv  string
		changes []nodesystem.Change
		want    nodesystem.Refusal
	}{{
		name:    "a key outside the table",
		changes: []nodesystem.Change{{Key: "seed", Value: nodesystem.Text("x")}},
		want:    nodesystem.Refusal{Key: "seed", Reason: nodesystem.UnknownSetting},
	}, {
		name:    "a key the environment sets",
		dotEnv:  "SLIPSTREAM_LOG_LEVEL=warn\n",
		changes: []nodesystem.Change{{Key: "log.level", Value: nodesystem.Text("debug")}},
		want:    nodesystem.Refusal{Key: "log.level", Reason: nodesystem.SetByEnvironment, Variable: "SLIPSTREAM_LOG_LEVEL"},
	}, {
		name: "one good change and one out of range",
		changes: []nodesystem.Change{
			{Key: "log.max_files", Value: nodesystem.Text("4")},
			{Key: "log.max_size_mb", Value: nodesystem.Number(1001)},
		},
		want: nodesystem.Refusal{Key: "log.max_size_mb", Reason: nodesystem.OutOfRange, Sentence: "1000"},
	}, {
		name:    "a value of the wrong form",
		changes: []nodesystem.Change{{Key: "log.compress", Value: nodesystem.Text("maybe")}},
		want:    nodesystem.Refusal{Key: "log.compress", Reason: nodesystem.OutOfRange},
	}, {
		name:    "the web port on the Node link's port",
		changes: []nodesystem.Change{{Key: "http.listen", Value: nodesystem.Number(8485)}},
		want:    nodesystem.Refusal{Key: "http.listen", Reason: nodesystem.PortUsedByNode, Sentence: "8485"},
	}, {
		name:    "the web address with no port",
		changes: []nodesystem.Change{{Key: "http.listen", Value: nodesystem.Word("[::1]")}},
		want:    nodesystem.Refusal{Key: "http.listen", Reason: nodesystem.OutOfRange},
	}, {
		name:    "the web address on port 0",
		changes: []nodesystem.Change{{Key: "http.listen", Value: nodesystem.Word("[::1]:0")}},
		want:    nodesystem.Refusal{Key: "http.listen", Reason: nodesystem.OutOfRange},
	}} {
		t.Run(c.name, func(t *testing.T) {
			f := testConfigFile(t, yaml, c.dotEnv)
			_, _, err := f.set(t.Context(), nil, c.changes)
			if !refusedAs(err, c.want) {
				t.Fatalf("err = %#v, want %#v", err, c.want)
			}
			if got := readConfigYAML(t, f); got != yaml {
				t.Fatalf("config.yaml = %q, want it unchanged", got)
			}
		})
	}
}

func TestSetRefusesALinkedConfigFile(t *testing.T) {
	f := testConfigFile(t, "", "")
	target := filepath.Join(shortTempDir(t), "config.yaml")
	writeFile(t, target, "name: media\n")
	if err := os.Symlink(target, filepath.Join(f.dir.Path(), "config.yaml")); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	_, _, err := f.set(t.Context(), nil, []nodesystem.Change{{Key: "log.level", Value: nodesystem.Text("debug")}})
	want := nodesystem.Refusal{Reason: nodesystem.NotWritable, Sentence: "config.yaml"}
	if !refusedAs(err, want) {
		t.Fatalf("err = %#v, want %#v", err, want)
	}
}

func TestSetConfigWritesThroughTheRunningNode(t *testing.T) {
	env, _ := testEnv(t)
	n := startTest(t, env)

	n.rootLog.Debug("before")
	level := "debug"
	keys, online, err := SetConfig(t.Context(), env, []ConfigChange{{Key: "log.level", Value: &level}})
	if err != nil || !online {
		t.Fatalf("set = online %t, %v; want online, nil", online, err)
	}
	if got := changedLine("log.level", keys, online); got != "log.level: debug" {
		t.Fatalf("line = %q", got)
	}
	n.rootLog.Debug("after")
	if got := debugRecords(t, env); !slices.Equal(got, []string{"after"}) {
		t.Fatalf("debug records = %v, want [after]", got)
	}

	ports := "3"
	keys, online, err = SetConfig(t.Context(), env, []ConfigChange{{Key: "http.ports_to_try", Value: &ports}})
	if err != nil {
		t.Fatal(err)
	}
	if got := changedLine("http.ports_to_try", keys, online); got != "http.ports_to_try: 3, applies after restart" {
		t.Fatalf("line = %q", got)
	}
	if got, want := readConfigYAML(t, env.configFile()), "log:\n  level: debug\nhttp:\n  ports_to_try: 3\n"; got != want {
		t.Fatalf("config.yaml = %q, want %q", got, want)
	}

	port := "[::]:9000"
	_, _, err = SetConfig(t.Context(), env, []ConfigChange{{Key: "http.listen", Value: &port}})
	want := nodesystem.Refusal{Key: "http.listen", Reason: nodesystem.SetByEnvironment, Variable: "SLIPSTREAM_HTTP_LISTEN"}
	if !refusedAs(err, want) {
		t.Fatalf("err = %#v, want %#v", err, want)
	}

	n.Stop()
	waitDone(t, n)
	keys, online, err = SetConfig(t.Context(), env, []ConfigChange{{Key: "log.level"}})
	if err != nil || online {
		t.Fatalf("set = online %t, %v; want offline, nil", online, err)
	}
	if got := changedLine("log.level", keys, online); got != "log.level: info" {
		t.Fatalf("line = %q", got)
	}
}

func TestAStoppedNodeJudgesFolderAndProgramRows(t *testing.T) {
	env, _ := testEnv(t)
	data := env.DataDir.Path()
	for _, d := range []string{filepath.Join("downloads", "kept"), "backups", "converting", "other"} {
		if err := os.MkdirAll(filepath.Join(data, d), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	program := filepath.Join(data, "other", "ffmpeg")
	writeFile(t, program, "#!/bin/sh\n")
	if err := os.Chmod(program, 0o700); err != nil { //nolint:gosec // a program to run needs its execute bit
		t.Fatal(err)
	}
	notes := filepath.Join(data, "other", "notes.txt")
	writeFile(t, notes, "x")
	notesRefusal := "Can't run this file because it isn't a program SlipStream can run."
	if runtime.GOOS == "windows" {
		notesRefusal = ""
	}
	for _, c := range []struct {
		key, value, refusal string
	}{
		{"backups.folder", filepath.Join(data, "downloads", "kept"), "Can't use this folder because it's inside the downloads folder."},
		{"converting.scratch", filepath.Join(data, "backups"), "Can't use this folder because it's inside the Backups folder."},
		{"converting.scratch", data, "Can't use this folder because it contains the downloads folder."},
		{"converting.scratch", filepath.Join(data, "converting"), ""},
		{"converting.ffmpeg", filepath.Join(data, "other", "gone"), "Can't run this file because it doesn't exist."},
		{"converting.ffmpeg", notes, notesRefusal},
		{"converting.ffmpeg", filepath.Join(data, "other"), "Can't run this file because it isn't a program SlipStream can run."},
		{"converting.ffmpeg", "ffmpeg", "Can't run this file because it isn't a full path."},
		{"converting.ffmpeg", program, ""},
	} {
		value := c.value
		keys, online, err := SetConfig(t.Context(), env, []ConfigChange{{Key: c.key, Value: &value}})
		if c.refusal != "" {
			if want := (nodesystem.Refusal{Key: c.key, Reason: nodesystem.UnusableFolder, Sentence: c.refusal}); !refusedAs(err, want) {
				t.Errorf("%s = %s: err = %v, want %q", c.key, c.value, err, c.refusal)
			}
			continue
		}
		if err != nil || online || changedLine(c.key, keys, online) != c.key+": "+c.value {
			t.Errorf("%s = %s: line %q, online %t, %v; want saved offline", c.key, c.value, changedLine(c.key, keys, online), online, err)
		}
	}
}

func debugRecords(t *testing.T, env Env) []string {
	t.Helper()
	logged, err := os.ReadFile(filepath.Join(env.DataDir.LogsDir(), "slipstream.log"))
	if err != nil {
		t.Fatal(err)
	}
	var msgs []string
	for _, r := range records(t, string(logged)) {
		if r["level"] == "DEBUG" {
			msgs = append(msgs, fmt.Sprint(r["msg"]))
		}
	}
	return msgs
}
