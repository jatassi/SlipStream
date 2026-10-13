package movie

import (
	ct "example.com/app/internal/contenttype"
	"example.com/app/internal/contenttypes/movie/internal/endpoints"
)

var Def = ct.Define[endpoints.Endpoints]("movie", nil, endpoints.DefaultEndpoints, nil)
