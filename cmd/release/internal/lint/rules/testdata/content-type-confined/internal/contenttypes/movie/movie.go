package movie

import ct "example.com/app/internal/contenttype"

type Endpoints struct{}

var Def = ct.Define[Endpoints]("movie", nil, nil, nil)
