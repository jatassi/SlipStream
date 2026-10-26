package sharing

import "example.com/app/internal/role"

var Def = role.WideDef[int, int, int]{Name: role.Wide("sharing")}
