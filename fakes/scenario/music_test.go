package scenario

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidateChecksTheMusicCatalog(t *testing.T) {
	const (
		artist = "a1000000-0000-4000-8000-000000000001"
		album  = "b1000000-0000-4000-8000-000000000001"
		rec    = "d1000000-0000-4000-8000-000000000001"
	)
	track := MusicTrack{Recording: rec, Title: "Tidewater", Disc: 1}
	good := MusicAlbum{ID: album, Release: "c1000000-0000-4000-8000-000000000001", Title: "Harbor Lights", Kind: "Album", Tracks: []MusicTrack{track}}
	bad := MusicAlbum{ID: "b1", Release: "c1000000-0000-4000-8000-000000000002", Title: "Lost", Kind: "Live", FirstRelease: "spring", Tracks: []MusicTrack{track, {Recording: rec, Title: "Again", Disc: 0}}}
	s := Scenario{Name: "x", Nodes: []Node{{Name: "home"}}, Music: &Music{
		Artists: []MusicArtist{
			{ID: artist, Revisions: []MusicRevision{{Name: "The Lanterns", SortName: "Lanterns, The", Albums: []MusicAlbum{good, good}}}},
			{ID: artist},
			{ID: "a2", Revisions: []MusicRevision{{Name: "Other", SortName: "Other", Albums: []MusicAlbum{good, bad}}}},
		},
	}}
	want := []string{
		"scenario x: music artist a1000000-0000-4000-8000-000000000001 revision 0 lists album b1000000-0000-4000-8000-000000000001 twice",
		"scenario x: music artist a1000000-0000-4000-8000-000000000001 is listed twice",
		"scenario x: music artist a1000000-0000-4000-8000-000000000001 has no revision",
		`scenario x: music artist "a2" is not a UUID`,
		"scenario x: music album b1000000-0000-4000-8000-000000000001 belongs to artists a1000000-0000-4000-8000-000000000001 and a2",
		`scenario x: music artist a2 revision 0 album "Lost" has an id "b1" that is not a UUID`,
		`scenario x: music artist a2 revision 0 album b1 kind "Live" is not one of Album, EP, Single`,
		`scenario x: music artist a2 revision 0 album b1 first_release "spring" is not YYYY, YYYY-MM or YYYY-MM-DD`,
		"scenario x: music artist a2 revision 0 album b1 lists recording d1000000-0000-4000-8000-000000000001 twice",
		"scenario x: music artist a2 revision 0 album b1 track d1000000-0000-4000-8000-000000000001 is on disc 0; discs start at 1",
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("Validate passed")
	}
	if got := strings.Split(err.Error(), "\n"); !reflect.DeepEqual(got, want) {
		t.Fatalf("Validate =\n%s\nwant\n%s", err, strings.Join(want, "\n"))
	}
}

func TestTheMusicScenarioReadsBackAsItself(t *testing.T) {
	s, err := Load("music")
	if err != nil {
		t.Fatal(err)
	}
	read, err := parse("music", s.YAML())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(read, s) || read.Hash() != s.Hash() {
		t.Fatalf("music read back as %+v, want %+v", read, s)
	}
	if got := s.ContentTypes(); !reflect.DeepEqual(got, []string{"music"}) {
		t.Fatalf("ContentTypes = %v, want [music]", got)
	}
}
