package mediaserverlink

import (
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
)

// Def is the Media Server link Role: the servers a Node reaches, its live
// view of them, and its reports.
var Def = role.Def[*msv1.Intent, *msv1.Facts, *msv1.Report]{Name: role.MediaServerLink}
