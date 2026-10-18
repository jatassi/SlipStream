package targets_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
)

func named(t *testing.T, cuts ...release.CutKey) targets.Editions {
	t.Helper()
	list, err := targets.NamedCuts(cuts...)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func TestWantedAppliesTheEditionRuleToEveryListShapeAndSighting(t *testing.T) {
	const extended, imax, theatrical release.CutKey = "extended", "imax", "theatrical"
	for _, c := range []struct {
		name string
		list targets.Editions
		seen []release.CutKey
		want []targets.Edition
	}{
		{"any cut, nothing seen", targets.AnyCut(), nil, []targets.Edition{targets.AnyEdition}},
		{"any cut, cuts seen", targets.AnyCut(), []release.CutKey{extended, imax}, []targets.Edition{targets.AnyEdition}},
		{"Standard alone", named(t, release.Standard), []release.CutKey{extended}, []targets.Edition{"standard"}},
		{"Standard and Extended, nothing seen", named(t, release.Standard, extended), nil, []targets.Edition{"standard"}},
		{"Standard and Extended, another cut seen", named(t, release.Standard, extended), []release.CutKey{imax}, []targets.Edition{"standard"}},
		{"Standard and Extended, Extended seen", named(t, release.Standard, extended), []release.CutKey{extended}, []targets.Edition{"standard", "extended"}},
		{"Extended before Standard, Extended seen", named(t, extended, release.Standard), []release.CutKey{extended}, []targets.Edition{"extended", "standard"}},
		{"Standard, Extended and IMAX, IMAX seen", named(t, release.Standard, extended, imax), []release.CutKey{imax, theatrical}, []targets.Edition{"standard", "imax"}},
		{"Extended alone, nothing seen", named(t, extended), nil, []targets.Edition{"extended"}},
		{"Extended and IMAX, nothing seen", named(t, extended, imax), nil, []targets.Edition{"extended", "imax"}},
		{"Extended and IMAX, Standard seen", named(t, extended, imax), []release.CutKey{release.Standard}, []targets.Edition{"extended", "imax"}},
	} {
		if got := targets.Wanted(c.list, c.seen); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Wanted = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestAnEditionTakesItsOwnCutAndAnyTakesEvery(t *testing.T) {
	if !targets.AnyEdition.Takes(release.Standard) || !targets.AnyEdition.Takes("extended") {
		t.Fatal("any doesn't take every cut")
	}
	if extended := targets.EditionOf("extended"); !extended.Takes("extended") || extended.Takes(release.Standard) {
		t.Fatal("Extended takes another cut than its own, or not its own")
	}
	if got := targets.AnyCut().Filling("extended"); got != targets.AnyEdition {
		t.Fatalf("an Extended file fills %s under any cut, want any", got)
	}
	if got := named(t, release.Standard, "extended").Filling("extended"); got != "extended" {
		t.Fatalf("an Extended file fills %s under named cuts, want extended", got)
	}
}

func TestANamedListNamesSomeCutsEachOnce(t *testing.T) {
	if _, err := targets.NamedCuts(); !errors.Is(err, targets.ErrNoCuts) {
		t.Fatalf("NamedCuts() = %v, want ErrNoCuts", err)
	}
	if _, err := targets.NamedCuts("extended", release.Standard, "extended"); !errors.Is(err, targets.ErrCutTwice) {
		t.Fatalf("NamedCuts naming Extended twice = %v, want ErrCutTwice", err)
	}
	if list := named(t, "extended", release.Standard); list.Any() || !reflect.DeepEqual(list.Cuts(), []release.CutKey{"extended", release.Standard}) {
		t.Fatalf("NamedCuts(extended, standard) = %v, want those cuts in order", list.Cuts())
	}
}
