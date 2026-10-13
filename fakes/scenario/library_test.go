package scenario

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidateChecksTheLibrary(t *testing.T) {
	s := Scenario{Name: "x", Nodes: []Node{{Name: "home"}}, Library: &Library{
		Roots: []LibraryRoot{
			{ContentType: "music", Folder: "music"},
			{ContentType: "music", Folder: "Music"},
			{ContentType: "fixture", Folder: "music"},
		},
		Targets: []LibraryTarget{
			{Name: "1080p", Resolutions: []string{"1080p"}},
			{Name: "1080p", Resolutions: []string{"1440p"}},
			{Name: "none"},
		},
		Profiles: []LibraryProfile{
			{Name: "Standard", ContentType: "music", Targets: []string{"1080p", "4k"}},
			{Name: "Standard", ContentType: "music"},
		},
	}}
	want := []string{
		"scenario x: library is set, but no node is controller",
		"scenario x: library has two roots for content type music",
		`scenario x: library root folder "Music" does not match ^[a-z0-9][a-z0-9._-]*$`,
		"scenario x: library has two roots in folder music",
		"scenario x: library has two targets named 1080p",
		`scenario x: library target 1080p: "1440p" is not a resolution`,
		"scenario x: library target none has no resolutions",
		"scenario x: library profile Standard names target 4k, which library.targets does not list",
		"scenario x: library has two profiles named Standard",
		"scenario x: library profile Standard has no targets",
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("Validate passed")
	}
	if got := strings.Split(err.Error(), "\n"); !reflect.DeepEqual(got, want) {
		t.Fatalf("Validate =\n%s\nwant\n%s", err, strings.Join(want, "\n"))
	}
	if got := s.ContentTypes(); !reflect.DeepEqual(got, []string{"fixture", "music"}) {
		t.Fatalf("ContentTypes = %v, want [fixture music]", got)
	}
}
