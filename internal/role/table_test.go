package role_test

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/role"
)

func TestEveryRoleIsListedOnceAndParses(t *testing.T) {
	names := role.Names()
	for i, n := range names {
		if slices.Contains(names[i+1:], n) {
			t.Fatalf("%q is listed twice", n)
		}
		got, err := role.ParseName(string(n))
		if err != nil || got != n {
			t.Fatalf("ParseName(%q) = %q, %v", n, got, err)
		}
		if n.Label() == "" || n.Purpose() == "" {
			t.Fatalf("%q has label %q and purpose %q", n, n.Label(), n.Purpose())
		}
	}
	if _, err := role.ParseName("probe"); err == nil {
		t.Fatal(`ParseName("probe") accepted the test Role`)
	}
}

func TestTheRuleColumnsMatchTheSpec(t *testing.T) {
	var always, controller, reach []role.Name
	for _, n := range role.Names() {
		r := n.Rule()
		if r.Always {
			always = append(always, n)
		}
		if r.Controller {
			controller = append(controller, n)
		}
		if r.Reach {
			reach = append(reach, n)
		}
		if want := map[role.Name]int{role.Relaying: 1}[n]; r.Max != want {
			t.Errorf("%s Max = %d, want %d", n, r.Max, want)
		}
	}
	if want := []role.Name{role.Searching, role.Downloading, role.Storage, role.Metadata, role.Notifying}; !slices.Equal(always, want) {
		t.Errorf("Always = %v, want %v", always, want)
	}
	if want := []role.Name{role.Serving}; !slices.Equal(controller, want) {
		t.Errorf("Controller = %v, want %v", controller, want)
	}
	if want := []role.Name{role.MediaServerLink, role.Notifying}; !slices.Equal(reach, want) {
		t.Errorf("Reach = %v, want %v", reach, want)
	}
	if got, want := role.Probes(), []role.Probe{role.EngineStarts, role.DataWritable, role.ListenerUp}; !slices.Equal(got, want) {
		t.Errorf("Probes() = %v, want %v", got, want)
	}
}

func TestSet(t *testing.T) {
	s := role.Of(role.Serving, role.Searching)
	if !s.Has(role.Searching) || !s.Has(role.Serving) || s.Has(role.Downloading) || s.Len() != 2 {
		t.Fatalf("Of(Serving, Searching) = %v", s.Names())
	}
	if got := s.Names(); !slices.Equal(got, []role.Name{role.Searching, role.Serving}) {
		t.Fatalf("Names() = %v, want table order", got)
	}
	s = s.With(role.Downloading).Without(role.Searching)
	if got := s.Names(); !slices.Equal(got, []role.Name{role.Downloading, role.Serving}) {
		t.Fatalf("after With and Without = %v", got)
	}
	if role.Of(role.Names()...).Len() != len(role.Names()) {
		t.Fatal("a Set can't hold every Role")
	}
	if s.Has("probe") {
		t.Fatal(`a Set has the test Role "probe"`)
	}
	defer func() {
		if recover() == nil {
			t.Fatal(`Of("probe") didn't panic`)
		}
	}()
	role.Of("probe")
}
