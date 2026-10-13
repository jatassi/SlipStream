package tv

import (
	"example.com/app/internal/contenttype"
	"example.com/app/internal/contenttypes/movie"
)

var Def = contenttype.Define[struct{}]("tv", nil, nil, movie.Def)
