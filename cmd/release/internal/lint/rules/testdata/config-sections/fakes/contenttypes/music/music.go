package music

import "example.com/app/internal/contenttype"

var Def = contenttype.Define("music", nil, DefaultEndpoints, nil)

type Timings struct{}

func DefaultTimings() Timings { return Timings{} }
