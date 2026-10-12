package world

import (
	"example.com/app/fakes/fake"
	"example.com/app/fakes/music"
	"example.com/app/fakes/tracker"
)

var fakes = []fake.Spec{
	music.Fake,
	tracker.Fake,
}
