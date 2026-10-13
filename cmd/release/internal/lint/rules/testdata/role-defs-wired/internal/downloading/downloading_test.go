package downloading

import "example.com/app/internal/role"

var Other = role.Def[int, int, int]{Name: role.Downloading}
