package downloadclients_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	tc "github.com/jatassi/slipstream/internal/torrent/clients"
)

// checked is the words column's CHECK in table lists, in order.
func checked(t *testing.T, lib library, table, column string) []string {
	t.Helper()
	var sql string
	if err := lib.QueryRowContext(t.Context(), "SELECT sql FROM sqlite_schema WHERE type = 'table' AND name = ?", table).Scan(&sql); err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`CHECK \(` + column + ` IN \(([^)]*)\)\)`).FindStringSubmatch(sql)
	if m == nil {
		t.Fatalf("%s has no CHECK on %s", table, column)
	}
	var words []string
	for w := range strings.SplitSeq(m[1], ",") {
		words = append(words, strings.Trim(strings.TrimSpace(w), "'"))
	}
	return words
}

func words[S ~string](values []S) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

func TestTheSchemaChecksListExactlyTheDomainValues(t *testing.T) {
	lib := open(t)
	for _, c := range []struct {
		table, column string
		want          []string
	}{
		{"downloadclients_torrent", "removed_reason", words(dc.RemoveReasons)},
		{"downloadclients_client", "kind", words(tc.Kinds)},
	} {
		if got := checked(t, lib, c.table, c.column); !slices.Equal(got, c.want) {
			t.Errorf("%s.%s's CHECK lists %v, want %v", c.table, c.column, got, c.want)
		}
	}
}
