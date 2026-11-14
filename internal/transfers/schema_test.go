package transfers

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/transfers/source"
)

// TestTheSchemaChecksEveryKindGoWrites pins each column's latest CHECK
// list in the library schema to the Go values written into it, so a value added on
// one side alone fails here.
func TestTheSchemaChecksEveryKindGoWrites(t *testing.T) {
	var schema []byte
	files, err := filepath.Glob("internal/ledger/schema/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		schema = append(schema, b...)
	}
	for column, want := range map[string][]string{
		"failure_code":     {string(SourceGone), string(SourceFailed), string(Damaged), string(SourceWrittenOff), string(Rerouted)},
		"destination_kind": {Root{}.kind()},
		"source_kind":      {string(source.TorrentKind), string(source.PathKind)},
	} {
		all := regexp.MustCompile(column+` IN \(([^)]*)\)`).FindAllSubmatch(schema, -1)
		if all == nil {
			t.Fatalf("the schema has no CHECK list for %s", column)
		}
		m := all[len(all)-1]
		var got []string
		for v := range strings.SplitSeq(string(m[1]), ",") {
			got = append(got, strings.Trim(strings.TrimSpace(v), "'"))
		}
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("%s checks %v, want the Go values %v", column, got, want)
		}
	}
}
