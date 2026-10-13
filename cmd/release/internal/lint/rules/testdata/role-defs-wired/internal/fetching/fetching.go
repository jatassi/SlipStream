package fetching

import "example.com/app/internal/role"

var (
	Def   = role.Def[int, int, int]{Name: role.Downloading}
	Named = role.Def[int, int, int]{Name: role.Name("fetching")}
)
