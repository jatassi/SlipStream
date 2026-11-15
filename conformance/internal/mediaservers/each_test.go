package mediaservers

import "testing"

func TestASkippedCaseLetsTheNextCaseRun(t *testing.T) {
	var ran []string
	Each(t, []Case{
		{Name: "first", Run: func(t *testing.T) { Skip(t, "no player"); ran = append(ran, "first") }},
		{Name: "second", Run: func(*testing.T) { ran = append(ran, "second") }},
	})
	if len(ran) != 1 || ran[0] != "second" {
		t.Fatalf("ran %v, want only second", ran)
	}
	if t.Skipped() {
		t.Fatal("a skipped case skipped the whole test")
	}
}
