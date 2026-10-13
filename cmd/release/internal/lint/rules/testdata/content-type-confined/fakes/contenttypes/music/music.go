package music

import (
	"example.com/app/fakes/contenttypes/music/internal/client"
	"example.com/app/internal/contenttype"
)

var Def = contenttype.Define[struct{}]("music", nil, nil, client.New)
