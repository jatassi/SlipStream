package grabs_test

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// checked is the values column's CHECK in table lists, in order.
func checked(t *testing.T, lib library, table, column string) []string {
	t.Helper()
	var sql string
	if err := lib.QueryRowContext(t.Context(), "SELECT sql FROM sqlite_schema WHERE type = 'table' AND name = ?", table).Scan(&sql); err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(column + ` (?:TEXT|INTEGER)[^,]*?CHECK \(` + column + ` IN \(([^)]*)\)\)`).FindStringSubmatch(sql)
	if m == nil {
		t.Fatalf("%s has no CHECK on %s", table, column)
	}
	var values []string
	for v := range strings.SplitSeq(m[1], ",") {
		values = append(values, strings.Trim(strings.TrimSpace(v), "'"))
	}
	return values
}

var queueTables = []string{"grabs_search", "grabs_nudge", "grabs_due"}

func TestLanesMatchTheCheck(t *testing.T) {
	lib := storagetest.Library(t, grabstest.Schemas()...)
	var want []string
	for _, l := range pace.Lanes {
		want = append(want, strconv.FormatInt(int64(l), 10))
	}
	for _, table := range queueTables {
		if got := checked(t, lib, table, "lane"); !slices.Equal(got, want) {
			t.Errorf("%s.lane's CHECK lists %v, want pace.Lanes %v", table, got, want)
		}
	}
}

// TestTriggersMatchTheCheck: a run never has retarget, which queues
// nothing, a nudge never has rss, which is a run's trigger only, and a
// queue row has neither.
func TestTriggersMatchTheCheck(t *testing.T) {
	lib := storagetest.Library(t, grabstest.Schemas()...)
	without := func(out ...searches.Trigger) []string {
		var ts []string
		for _, tr := range searches.Triggers() {
			if !slices.Contains(out, tr) {
				ts = append(ts, string(tr))
			}
		}
		return ts
	}
	for table, want := range map[string][]string{
		"grabs_search": without(searches.Retarget),
		"grabs_nudge":  without(searches.RSS),
		"grabs_due":    without(searches.Retarget, searches.RSS),
	} {
		if got := checked(t, lib, table, "trigger"); !slices.Equal(got, want) {
			t.Errorf("%s.trigger's CHECK lists %v, want %v", table, got, want)
		}
	}
}
