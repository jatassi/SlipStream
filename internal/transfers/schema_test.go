package transfers

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/transfers/source"
)

// TestTheSchemaChecksEveryKindGoWrites pins each CHECK list in the
// library schema to the Go values written into it, so a value added on
// one side alone fails here.
func TestTheSchemaChecksEveryKindGoWrites(t *testing.T) {
	schema, err := os.ReadFile("internal/ledger/schema/00001_transfers.sql")
	if err != nil {
		t.Fatal(err)
	}
	for column, want := range map[string][]string{
		"failure_code":     {string(SourceGone), string(SourceFailed), string(Damaged), string(SourceWrittenOff), string(Rerouted)},
		"destination_kind": {Root{}.kind()},
		"source_kind":      {string(source.TorrentKind)},
	} {
		m := regexp.MustCompile(column + ` IN \(([^)]*)\)`).FindSubmatch(schema)
		if m == nil {
			t.Fatalf("the schema has no CHECK list for %s", column)
		}
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
