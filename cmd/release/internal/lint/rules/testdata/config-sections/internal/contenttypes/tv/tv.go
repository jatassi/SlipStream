package tv

type Endpoints struct{}

func DefaultEndpoints() Endpoints { return Endpoints{} }

func define(id string, migrations any, defaultEndpoints func() Endpoints, open any) int { return 0 }

var Def = define("tv", nil, DefaultEndpoints, nil)
