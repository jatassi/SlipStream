package indexers

import (
	"archive/zip"
	"bytes"
	"cmp"
	"context"
	"crypto/sha1" //nolint:gosec // G505: a git blob id is SHA-1, checked for integrity and not for secrecy
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/storage"
)

// CatalogSchema is the Cardigann schema version SlipStream reads. The
// catalog serves every older file under it by fallback.
const CatalogSchema = 12

// maxCatalogBody bounds what one catalog response may hold: the index and
// package.zip are each under 2 MB today.
const maxCatalogBody = 64 << 20

// IndexEntry is one definition the catalog's index lists.
type IndexEntry struct {
	ID          string `json:"id"`
	File        string `json:"file"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Language    string `json:"language"`
	// SHA is the git blob id of the file's bytes.
	SHA string `json:"sha"`
}

// catalogSource is one catalog address, read over client.
type catalogSource struct {
	client *http.Client
	base   string
	// share is Timings.PackageShare.
	share int
	// request is Timings.CatalogRequest.
	request time.Duration
}

// fetched is one read of a catalog: its index, or none when the index is
// unchanged since the read that ended with lastModified.
type fetched struct {
	unchanged    bool
	lastModified string
	index        []IndexEntry
	// files are the files fetch was asked for, by file, each checked
	// against its sha.
	files map[string][]byte
}

// fetch reads the index, sending since as If-Modified-Since unless it is
// empty, then the files of every entry want accepts: from package.zip when
// they are more than share percent of the index, else one by one. A file
// the zip lacks or holds at another sha is fetched alone.
func (c catalogSource) fetch(ctx context.Context, since string, want func(IndexEntry) bool) (fetched, error) {
	res, body, err := c.get(ctx, "", since)
	if err != nil {
		return fetched{}, err
	}
	if res.StatusCode == http.StatusNotModified {
		return fetched{unchanged: true, lastModified: since}, nil
	}
	index, err := parseIndex(body)
	if err != nil {
		return fetched{}, fmt.Errorf("read the index at %s: %w", c.url(""), err)
	}
	out := fetched{lastModified: res.Header.Get("Last-Modified"), index: index, files: map[string][]byte{}}
	var wanted []IndexEntry
	for _, e := range index {
		if want(e) {
			wanted = append(wanted, e)
		}
	}
	if len(wanted)*100 > c.share*len(index) {
		zipped, err := c.unzip(ctx, wanted)
		if err != nil {
			return fetched{}, err
		}
		out.files = zipped
	}
	for _, e := range wanted {
		if _, ok := out.files[e.File]; ok {
			continue
		}
		_, src, err := c.get(ctx, e.File, "")
		if err != nil {
			return fetched{}, err
		}
		if blobSHA(src) != e.SHA {
			return fetched{}, fmt.Errorf("%s doesn't match the sha its index lists", c.url(e.File))
		}
		out.files[e.File] = src
	}
	return out, nil
}

// unzip is the files of wanted that package.zip holds at their sha. A zip
// that can't be read gives none.
func (c catalogSource) unzip(ctx context.Context, wanted []IndexEntry) (map[string][]byte, error) {
	_, body, err := c.get(ctx, "package.zip", "")
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return out, nil //nolint:nilerr // each file is then fetched alone
	}
	entries := map[string]*zip.File{}
	for _, f := range archive.File {
		entries[f.Name] = f
	}
	for _, e := range wanted {
		f, ok := entries[e.File+".yml"]
		if !ok {
			continue
		}
		src, err := readZipped(f)
		if err == nil && blobSHA(src) == e.SHA {
			out[e.File] = src
		}
	}
	return out, nil
}

func readZipped(f *zip.File) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(io.LimitReader(r, maxCatalogBody))
}

func (c catalogSource) url(path string) string {
	return strings.TrimSuffix(c.base, "/") + "/" + strconv.Itoa(CatalogSchema) + "/" + path
}

// get fetches path under the schema's folder. Any status but 200 and a 304
// to since is an error.
func (c catalogSource) get(ctx context.Context, path, since string) (*http.Response, []byte, error) {
	u := c.url(path)
	ctx, cancel := context.WithTimeout(ctx, c.request)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, nil, fmt.Errorf("build the request for %s: %w", u, err)
	}
	if since != "" {
		req.Header.Set("If-Modified-Since", since)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request %s: %w", u, err)
	}
	defer func() { _ = res.Body.Close() }()
	switch {
	case res.StatusCode == http.StatusNotModified && since != "":
		return res, nil, nil
	case res.StatusCode == http.StatusNotFound && path == "":
		return nil, nil, fmt.Errorf("%s has no schema %d index", c.base, CatalogSchema)
	case res.StatusCode != http.StatusOK:
		return nil, nil, fmt.Errorf("%s responded %s", u, res.Status)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxCatalogBody))
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", u, err)
	}
	return res, body, nil
}

func parseIndex(body []byte) ([]IndexEntry, error) {
	var index []IndexEntry
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("it isn't a list of definitions: %w", err)
	}
	ids, files := map[string]bool{}, map[string]bool{}
	for _, e := range index {
		switch {
		case e.ID == "" || e.File == "" || e.SHA == "":
			return nil, errors.New("an entry has no id, file or sha")
		case strings.ContainsAny(e.File, "/\\"):
			return nil, fmt.Errorf("the file name %q has a path in it", e.File)
		case ids[e.ID] || files[e.File]:
			return nil, fmt.Errorf("it lists %s twice", e.ID)
		}
		ids[e.ID], files[e.File] = true, true
	}
	return index, nil
}

// blobSHA is the git blob id of src, the sha the catalog's index lists.
func blobSHA(src []byte) string {
	h := sha1.New() //nolint:gosec // G401: see the import
	_, _ = fmt.Fprintf(h, "blob %d\x00", len(src))
	_, _ = h.Write(src)
	return hex.EncodeToString(h.Sum(nil))
}

// ReadCatalog fetches every definition the schema 12 catalog at base
// lists, each checked against its sha, by file.
func ReadCatalog(ctx context.Context, client *http.Client, base string) ([]IndexEntry, map[string][]byte, error) {
	t := DefaultTimings()
	f, err := catalogSource{client: client, base: base, share: t.PackageShare, request: t.CatalogRequest}.fetch(ctx, "", func(IndexEntry) bool { return true })
	return f.index, f.files, err
}

// definitionKey names one definition version.
type definitionKey struct{ id, sha string }

// refresh is the catalog refresh task's one unit. It reads the index with
// If-Modified-Since and fetches each changed file. In one Write it stores
// every new version with what Load made of it, lists the index as the
// catalog, moves every pin whose new version loads and drops the versions
// no one lists or pins. Any failure keeps the last good catalog.
func (x *Indexers) refresh(ctx context.Context) error {
	var state db.IndexersCatalogState
	have := map[definitionKey]bool{}
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		var err error
		if state, err = q.GetCatalogState(ctx); err != nil {
			return err
		}
		keys, err := q.ListDefinitionKeys(ctx)
		for _, k := range keys {
			have[definitionKey{string(k.ID), k.Sha}] = true
		}
		return err
	})
	if err != nil {
		return err
	}
	since := deref(state.LastModified)
	f, err := x.catalogSource(deref(state.Address)).fetch(ctx, since, func(e IndexEntry) bool { return !have[definitionKey{e.ID, e.SHA}] })
	if err != nil {
		return err
	}
	now := storage.Now()
	var moved []ID
	err = x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if f.unchanged {
			return q.SetCatalogRefreshed(ctx, db.SetCatalogRefreshedParams{RefreshedAt: now, LastModified: orNil(since)})
		}
		for _, e := range f.index {
			if src, ok := f.files[e.File]; ok {
				if err := q.InsertDefinition(ctx, definitionRow(e.ID, e.SHA, e.File, FromCatalog, src, now)); err != nil {
					return err
				}
			}
		}
		if err := listCatalog(ctx, q, f.index); err != nil {
			return err
		}
		var err error
		if moved, err = x.movePins(ctx, tx, now); err != nil {
			return err
		}
		if err := q.PruneDefinitions(ctx); err != nil {
			return err
		}
		return q.SetCatalogRefreshed(ctx, db.SetCatalogRefreshedParams{RefreshedAt: now, LastModified: orNil(f.lastModified)})
	})
	if err != nil {
		return err
	}
	x.catalogChanged(ctx)
	for _, id := range moved {
		x.changed(ctx, id)
	}
	return nil
}

func (x *Indexers) catalogSource(address string) catalogSource {
	return catalogSource{
		client: x.cfg.HTTP, base: cmp.Or(address, x.cfg.Endpoints.Catalog.String()),
		share: x.cfg.Timings.PackageShare, request: x.cfg.Timings.CatalogRequest,
	}
}

// catalogChanged publishes CatalogChanged with the last refresh's time.
func (x *Indexers) catalogChanged(ctx context.Context) {
	var state db.IndexersCatalogState
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		state, err = db.New(r).GetCatalogState(ctx)
		return err
	})
	if err != nil {
		x.cfg.Log.Error("read the Indexer catalog's state", "err", err)
		return
	}
	x.cfg.Catalogs.Publish(ctx, CatalogChanged{RefreshedAt: when(state.RefreshedAt.Time())})
}

func when(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// definitionRow is src as indexers_definition holds it: the summary Load
// made, or why it refused.
func definitionRow(id, sha, file string, origin Origin, src []byte, fetched storage.Time) db.InsertDefinitionParams {
	p := db.InsertDefinitionParams{ID: db.DefinitionID(id), Sha: sha, File: file, Origin: string(origin), Yaml: src, FetchedAt: fetched}
	d, err := cardigann.Load(src)
	if err != nil {
		reason := err.Error()
		p.Refusal = &reason
		return p
	}
	summary, err := json.Marshal(d.Summary())
	if err != nil {
		panic("indexers: a definition summary doesn't encode: " + err.Error())
	}
	s := string(summary)
	p.Summary = &s
	return p
}

// listCatalog replaces the catalog with index, each entry described by
// its stored version.
func listCatalog(ctx context.Context, q *db.Queries, index []IndexEntry) error {
	if err := q.ClearCatalog(ctx); err != nil {
		return err
	}
	for _, e := range index {
		def, err := q.GetDefinition(ctx, db.GetDefinitionParams{ID: db.DefinitionID(e.ID), Sha: e.SHA})
		if err != nil {
			return fmt.Errorf("read definition %s: %w", e.ID, err)
		}
		var categories []newznab.ID
		if s, ok, err := summaryOf(def); err != nil {
			return err
		} else if ok {
			categories = s.Caps.Categories
		}
		cats, err := json.Marshal(append([]newznab.ID{}, categories...))
		if err != nil {
			return fmt.Errorf("encode the categories: %w", err)
		}
		err = q.InsertCatalogEntry(ctx, db.InsertCatalogEntryParams{
			ID: db.DefinitionID(e.ID), File: e.File, Sha: e.SHA, Name: e.Name, Description: e.Description,
			Privacy: string(privacyOf(e.Type)), Language: e.Language, Categories: string(cats),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// privacyOf reads a definition's type as Prowlarr does: anything other
// than private or public is semi-private.
func privacyOf(t string) cardigann.Privacy {
	if p := cardigann.Privacy(t); p == cardigann.Private || p == cardigann.Public {
		return p
	}
	return cardigann.SemiPrivate
}

// summaryOf is what Load made of def, false when it refused def.
func summaryOf(def db.IndexersDefinition) (cardigann.Summary, bool, error) {
	if def.Summary == nil {
		return cardigann.Summary{}, false, nil
	}
	var s cardigann.Summary
	if err := decode(*def.Summary, &s); err != nil {
		return cardigann.Summary{}, false, fmt.Errorf("definition %s: %w", def.ID, err)
	}
	return s, true, nil
}
