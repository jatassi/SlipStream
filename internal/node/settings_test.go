package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node/internal/config"
	"github.com/jatassi/slipstream/internal/nodesystem"
)

func defaultView(t *testing.T, kind install.Kind) view {
	t.Helper()
	cfg, keys, err := loadConfig(fstest.MapFS{}, nil, noTypes, kind)
	if err != nil {
		t.Fatal(err)
	}
	return view{saved: &cfg, running: &cfg, defaults: &cfg, keys: keysByName(keys)}
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
	want := `[` +
		`{"key":"http.listen","label":"Port","source":"default","applies":"restart","isDefault":true,"pendingRestart":false,"kind":"port","value":8484,"running":8484,"default":8484},` +
		`{"key":"http.ports_to_try","label":"Ports to try","source":"default","applies":"restart","isDefault":true,"pendingRestart":false,"kind":"count","value":1,"running":1,"default":1,"min":1,"max":100,"unit":"ports"},` +
		`{"key":"log.level","label":"Level","source":"default","applies":"live","isDefault":true,"pendingRestart":false,"kind":"choice","value":"info","running":"info","default":"info","options":[{"value":"debug","label":"Debug"},{"value":"info","label":"Info"},{"value":"warn","label":"Warn"},{"value":"error","label":"Error"}]},` +
		`{"key":"log.max_size_mb","label":"Rotate at","source":"default","applies":"live","isDefault":true,"pendingRestart":false,"kind":"count","value":10,"running":10,"default":10,"min":1,"max":1000,"unit":"MB"},` +
		`{"key":"log.max_files","label":"Files kept","source":"default","applies":"live","isDefault":true,"pendingRestart":false,"kind":"count","value":5,"running":5,"default":5,"min":0,"max":100,"unit":"files","zero":"No limit"},` +
		`{"key":"log.max_age_days","label":"Days kept","source":"default","applies":"live","isDefault":true,"pendingRestart":false,"kind":"count","value":30,"running":30,"default":30,"min":0,"max":3650,"unit":"days","zero":"No limit"},` +
		`{"key":"log.compress","label":"Compress rotated files","source":"default","applies":"live","isDefault":true,"pendingRestart":false,"kind":"switch","value":true,"running":true,"default":true},` +
		`{"key":"log.viewer_lines","label":"Lines kept for the viewer","source":"default","applies":"live","isDefault":true,"pendingRestart":false,"kind":"count","value":2000,"running":2000,"default":2000,"min":100,"max":50000,"unit":"lines"}` +
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
	want := `{"key":"http.ports_to_try","label":"Ports to try","source":".env","applies":"restart","isDefault":false,"lockedBy":"SLIPSTREAM_HTTP_PORTS_TO_TRY","pendingRestart":true,"kind":"count","value":3,"running":10,"default":10,"min":1,"max":100,"unit":"ports"}`
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
	saved, keys, err := f.set([]nodesystem.Change{{Key: "log.level", Value: nodesystem.Text("debug")}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := readConfigYAML(t, f), "# kept\nname: media\nlog:\n  level: debug\n"; got != want {
		t.Fatalf("config.yaml = %q, want %q", got, want)
	}
	if k := keysByName(keys)["log.level"]; saved.Log.Level != "debug" || k.Source != config.SourceYAML {
		t.Fatalf("log.level = %s from %s, want debug from config.yaml", saved.Log.Level, k.Source)
	}

	saved, keys, err = f.set([]nodesystem.Change{{Key: "log.level", Value: nodesystem.Reset()}})
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
		want:    nodesystem.Refusal{Key: "seed", Reason: nodesystem.UnknownSetting, Sentence: "No setting has this key."},
	}, {
		name:    "a key the environment sets",
		dotEnv:  "SLIPSTREAM_LOG_LEVEL=warn\n",
		changes: []nodesystem.Change{{Key: "log.level", Value: nodesystem.Text("debug")}},
		want:    nodesystem.Refusal{Key: "log.level", Reason: nodesystem.SetByEnvironment, Sentence: "Set by SLIPSTREAM_LOG_LEVEL.", Variable: "SLIPSTREAM_LOG_LEVEL"},
	}, {
		name: "one good change and one out of range",
		changes: []nodesystem.Change{
			{Key: "log.max_files", Value: nodesystem.Text("4")},
			{Key: "log.max_size_mb", Value: nodesystem.Number(1001)},
		},
		want: nodesystem.Refusal{Key: "log.max_size_mb", Reason: nodesystem.OutOfRange, Sentence: "Between 1 and 1000 MB."},
	}, {
		name:    "a value of the wrong form",
		changes: []nodesystem.Change{{Key: "log.compress", Value: nodesystem.Text("maybe")}},
		want:    nodesystem.Refusal{Key: "log.compress", Reason: nodesystem.OutOfRange, Sentence: "On or off: true or false."},
	}, {
		name:    "the web port on the Node link's port",
		changes: []nodesystem.Change{{Key: "http.listen", Value: nodesystem.Number(8485)}},
		want:    nodesystem.Refusal{Key: "http.listen", Reason: nodesystem.PortUsedByNode, Sentence: "The Node link already uses port 8485."},
	}} {
		t.Run(c.name, func(t *testing.T) {
			f := testConfigFile(t, yaml, c.dotEnv)
			_, _, err := f.set(c.changes)
			if refused, ok := errors.AsType[*nodesystem.Refusal](err); !ok || *refused != c.want {
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
	_, _, err := f.set([]nodesystem.Change{{Key: "log.level", Value: nodesystem.Text("debug")}})
	want := nodesystem.Refusal{Reason: nodesystem.NotWritable, Sentence: "SlipStream can't rewrite config.yaml, a link or a mounted file. Edit it by hand."}
	if refused, ok := errors.AsType[*nodesystem.Refusal](err); !ok || *refused != want {
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
	want := nodesystem.Refusal{Key: "http.listen", Reason: nodesystem.SetByEnvironment, Sentence: "Set by SLIPSTREAM_HTTP_LISTEN.", Variable: "SLIPSTREAM_HTTP_LISTEN"}
	if refused, ok := errors.AsType[*nodesystem.Refusal](err); !ok || *refused != want {
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
