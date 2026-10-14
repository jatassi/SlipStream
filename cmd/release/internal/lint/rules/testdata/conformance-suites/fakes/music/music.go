package music

import "example.com/app/fakes/fake"

var Fake = fake.Spec{
	Name:   "music-meta",
	Online: fake.OnlineSuite("musicbrainz"),
}
