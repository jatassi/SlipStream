package mediaserverlink

import (
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
)

// Def is the Media Server link Role: the servers a Node reaches, its live
// view of them, and its reports. Only a change in how many of its servers
// respond is notable, which the Diagram's television shows.
var Def = role.Def[*msv1.Intent, *msv1.Facts, *msv1.Report]{
	Name:    role.MediaServerLink,
	Notable: func(last, next *msv1.Facts) bool { return responding(last) != responding(next) },
}

func responding(f *msv1.Facts) int {
	n := 0
	for _, s := range f.GetServers() {
		if s.GetResponding() {
			n++
		}
	}
	return n
}
