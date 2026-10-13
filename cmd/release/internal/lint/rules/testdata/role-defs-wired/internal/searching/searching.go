package searching

import "example.com/app/internal/role"

var Def = role.Def[*Intent, *Facts, *Result]{Name: role.Searching}

type Intent struct{}
type Facts struct{}
type Result struct{}
