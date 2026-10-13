package world

import (
	"github.com/jatassi/slipstream/fakes/contenttypes/music/musicmeta"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/tmdb"
)

var fakes = []fake.Spec{musicmeta.Fake, tmdb.Fake}
