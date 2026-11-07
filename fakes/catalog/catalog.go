// Package catalog is fake-catalog: Prowlarr's Indexer catalog in its real
// layout, an index of definitions with their git blob shas, each file
// alone and package.zip, serving the fictional sites' definitions from
// fakes/site/definitions. It answers If-Modified-Since with 304 until a
// cue changes what it lists.
package catalog

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // G505: a git blob id is SHA-1
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site/definitions"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

var (
	// Update lists Larkspur's second version, which loads.
	Update = fake.CueSpec{Name: "indexer-catalog-update"}
	// BadUpdate lists Larkspur's third version, which doesn't load.
	BadUpdate = fake.CueSpec{Name: "indexer-catalog-bad-update"}
	// Remove drops {"id": "<id>"} from the catalog.
	Remove = fake.CueSpec{Name: "indexer-catalog-remove"}
	// Schema serves only schema {"schema": <n>} from now on, so a client
	// that reads another finds no index.
	Schema = fake.CueSpec{Name: "indexer-catalog-schema"}
	// CorruptZip serves a package.zip that isn't a zip.
	CorruptZip = fake.CueSpec{Name: "indexer-catalog-corrupt-zip"}
	// StaleZip keeps package.zip holding the files as they are now while
	// later cues change the index and the single files, as the real
	// catalog's cache can.
	StaleZip = fake.CueSpec{Name: "indexer-catalog-stale-zip"}
)

// Fake is reached through endpoints.indexers.catalog. No Admin types its
// address unless they change the catalog address.
var Fake = fake.Spec{
	Name: "indexer-catalog",
	TLS:  fake.FrontTLS,
	New:  newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.Indexers.Catalog = endpoint.MustParseURL(at.URL(Base))
	},
	Online: fake.OnlineSuite("indexercatalog"),
	Cues:   []fake.CueSpec{Update, BadUpdate, Remove, Schema, CorruptZip, StaleZip},
}

// Base is the catalog's base path, to which a client adds the schema.
const Base = "/master"

// served is the schema SlipStream reads, the one the fake serves until a
// cue changes it.
const served = 12

// version is a later version of a definition: <id>-v<n>.yml.
var version = regexp.MustCompile(`-v\d+\.yml$`)

type server struct {
	mu     sync.Mutex
	schema int
	// files are the definitions listed, by file name, which is each one's
	// id.
	files map[string][]byte
	// zipped is what package.zip holds, nil while it holds files.
	zipped   map[string][]byte
	corrupt  bool
	modified time.Time
}

func newServer(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
	srv := &server{schema: served, files: map[string][]byte{}, modified: time.Now().UTC().Truncate(time.Second)}
	names, err := fs.Glob(definitions.FS, "*.yml")
	if err != nil {
		return nil, fmt.Errorf("list the definitions: %w", err)
	}
	for _, name := range names {
		if version.MatchString(name) {
			continue
		}
		if srv.files[strings.TrimSuffix(name, ".yml")], err = fs.ReadFile(definitions.FS, name); err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
	}
	for _, c := range []struct {
		spec fake.CueSpec
		play func(context.Context, struct{}) (any, error)
	}{
		{Update, srv.swap("larkspur", "larkspur-v2.yml")},
		{BadUpdate, srv.swap("larkspur", "larkspur-v3.yml")},
		{CorruptZip, srv.change(func() { srv.corrupt = true })},
		{StaleZip, srv.change(func() { srv.zipped = maps.Clone(srv.files) })},
	} {
		if err := fake.OnCue(env, c.spec.Name, c.play); err != nil {
			return nil, err
		}
	}
	if err := fake.OnCue(env, Remove.Name, srv.remove); err != nil {
		return nil, err
	}
	if err := fake.OnCue(env, Schema.Name, srv.serve); err != nil {
		return nil, err
	}
	return fake.HTTP(env, fake.Routes{
		"index":   {Pattern: "GET " + Base + "/{schema}/{$}", Handler: srv.index},
		"package": {Pattern: "GET " + Base + "/{schema}/package.zip", Handler: srv.pack},
		"file":    {Pattern: "GET " + Base + "/{schema}/{file}", Handler: srv.file},
	}), nil
}

// Changed is when the catalog last changed, its Last-Modified.
type Changed struct {
	Modified time.Time `json:"modified"`
}

// change runs f under the lock and moves Last-Modified on by at least a
// second, so the next If-Modified-Since sees the change.
func (s *server) change(f func()) func(context.Context, struct{}) (any, error) {
	return func(context.Context, struct{}) (any, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		f()
		s.modified = s.modified.Add(time.Second)
		if now := time.Now().UTC().Truncate(time.Second); now.After(s.modified) {
			s.modified = now
		}
		return Changed{Modified: s.modified}, nil
	}
}

func (s *server) swap(file, with string) func(context.Context, struct{}) (any, error) {
	return s.change(func() {
		src, err := fs.ReadFile(definitions.FS, with)
		if err != nil {
			panic("fake-catalog: " + with + " isn't embedded")
		}
		s.files[file] = src
	})
}

type removed struct {
	ID string `json:"id"`
}

func (s *server) remove(ctx context.Context, a removed) (any, error) {
	s.mu.Lock()
	_, ok := s.files[a.ID]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("the catalog lists no %s", a.ID)
	}
	return s.change(func() { delete(s.files, a.ID) })(ctx, struct{}{})
}

type schema struct {
	Schema int `json:"schema"`
}

func (s *server) serve(ctx context.Context, a schema) (any, error) {
	return s.change(func() { s.schema = a.Schema })(ctx, struct{}{})
}

// listed answers false, having written 404, for a schema the fake doesn't
// serve. s.mu is held.
func (s *server) listed(w http.ResponseWriter, r *http.Request) bool {
	if r.PathValue("schema") != strconv.Itoa(s.schema) {
		http.NotFound(w, r)
		return false
	}
	return true
}

func (s *server) index(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.listed(w, r) {
		return
	}
	w.Header().Set("Last-Modified", s.modified.Format(http.TimeFormat))
	if since, err := http.ParseTime(r.Header.Get("If-Modified-Since")); err == nil && !s.modified.After(since) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	entries := make([]entry, 0, len(s.files))
	for _, file := range slices.Sorted(maps.Keys(s.files)) {
		e, err := entryOf(file, s.files[file])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		entries = append(entries, e)
	}
	body, err := json.Marshal(entries)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(body)
}

func (s *server) file(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.listed(w, r) {
		return
	}
	src, ok := s.files[r.PathValue("file")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/yaml")
	_, _ = w.Write(src)
}

func (s *server) pack(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.listed(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	if s.corrupt {
		_, _ = w.Write([]byte("this is not a zip archive"))
		return
	}
	files := s.files
	if s.zipped != nil {
		files = s.zipped
	}
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for _, file := range slices.Sorted(maps.Keys(files)) {
		f, err := z.Create(file + ".yml")
		if err == nil {
			_, err = f.Write(files[file])
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := z.Close(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(buf.Bytes())
}

// entry is one definition as the catalog's index lists it: its metadata
// and settings without the request blocks.
type entry struct {
	ID             string         `json:"id"`
	File           string         `json:"file"`
	Implementation string         `json:"implementation"`
	Name           string         `json:"name"`
	Protocol       string         `json:"protocol"`
	Description    string         `json:"description"`
	Type           string         `json:"type"`
	Language       string         `json:"language"`
	Links          []string       `json:"links"`
	LegacyLinks    []string       `json:"legacylinks,omitempty"`
	Login          map[string]any `json:"login,omitempty"`
	Settings       []any          `json:"settings,omitempty"`
	SHA            string         `json:"sha"`
	Caps           any            `json:"caps"`
}

type definition struct {
	ID          string         `yaml:"id"`
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Type        string         `yaml:"type"`
	Language    string         `yaml:"language"`
	Links       []string       `yaml:"links"`
	LegacyLinks []string       `yaml:"legacylinks"`
	Login       map[string]any `yaml:"login"`
	Settings    []any          `yaml:"settings"`
	Caps        any            `yaml:"caps"`
}

func entryOf(file string, src []byte) (entry, error) {
	var d definition
	if err := yaml.Unmarshal(src, &d); err != nil {
		return entry{}, fmt.Errorf("read %s: %w", file, err)
	}
	e := entry{
		ID: d.ID, File: file, Implementation: "Cardigann", Name: d.Name, Protocol: "torrent", Description: d.Description, Type: d.Type,
		Language: d.Language, Links: d.Links, LegacyLinks: d.LegacyLinks, Settings: d.Settings, SHA: blobSHA(src), Caps: d.Caps,
	}
	if d.Login != nil {
		e.Login = map[string]any{}
		if c, ok := d.Login["captcha"]; ok {
			e.Login["captcha"] = c
		}
	}
	return e, nil
}

func blobSHA(src []byte) string {
	h := sha1.New() //nolint:gosec // G401: see the import
	_, _ = fmt.Fprintf(h, "blob %d\x00", len(src))
	_, _ = h.Write(src)
	return hex.EncodeToString(h.Sum(nil))
}
