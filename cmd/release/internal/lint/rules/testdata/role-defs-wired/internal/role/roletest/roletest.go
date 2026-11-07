package roletest

import "example.com/app/internal/role"

var Name = role.Name("probe")

var Def = role.Def[int, int, int]{Name: Name}

var WideDef = role.WideDef[int, int, int]{Name: role.Wide("wide-probe")}
