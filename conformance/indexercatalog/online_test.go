//go:build online

package indexercatalog_test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/indexercatalog"
	"github.com/jatassi/slipstream/internal/indexers"
)

// TestProwlarrsCatalogConforms reads Prowlarr's live catalog and keeps
// each definition in the fixture cache, where the cardigann package's
// catalog tests read it through SLIPSTREAM_CARDIGANN_CATALOG.
func TestProwlarrsCatalogConforms(t *testing.T) {
	r := conformance.Run(t, conformance.Target{Base: indexers.DefaultEndpoints().Catalog.String(), Client: &http.Client{Timeout: 5 * time.Minute}})
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(cache, "slipstream-fixtures", "indexer-catalog")
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	for file, src := range r.Files {
		if err := os.WriteFile(filepath.Join(dir, file+".yml"), src, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("cached %d definitions in %s", len(r.Files), dir)
}
