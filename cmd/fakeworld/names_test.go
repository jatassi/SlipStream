package main

import (
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
)

func TestCueAndCaptureNamesAreUnique(t *testing.T) {
	t.Parallel()
	taken := map[string]string{}
	claim := func(name, what string) {
		if had, ok := taken[name]; ok {
			t.Errorf("%s %s shares its name with a %s", what, name, had)
		}
		taken[name] = what
	}
	for verb := range verbs {
		taken[verb] = "verb"
	}
	for _, c := range world.Captures() {
		claim(c.Name, "capture")
	}
	for _, c := range world.Cues() {
		claim(c.Name, "cue")
	}
}
