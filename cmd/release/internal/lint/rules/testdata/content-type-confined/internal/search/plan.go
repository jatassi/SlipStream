package search

// Plan searches every Content Type, music among them, through its facet.
import (
	"strings"

	"example.com/app/fakes/contenttypes/music"
	"example.com/app/fakes/contenttypes/music/musicmeta"
)

var _ = strings.ToLower

var registered = music.Def

var _ = musicmeta.Fake
