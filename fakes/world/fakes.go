package world

import (
	"github.com/jatassi/slipstream/fakes/contenttypes/music/musicmeta"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/tmdb"
	"github.com/jatassi/slipstream/fakes/tracker"
)

var fakes = []fake.Spec{musicmeta.Fake, tmdb.Fake, tracker.Fake, site.Fake}
