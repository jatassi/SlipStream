// Package emby is fake-emby: one Emby server over the shared Media Server
// core, the same server at every Node's front. It copies Emby 4.10.1.0
// (#71, #101): its own routes (/Library/VirtualFolders/Query, /Items?Path=
// matched exactly, /Users/{user}/Items/{id}/UserData,
// /Users/{user}/PlayedItems/{id}, /embywebsocket), path reports under one
// 90 s timer that a repeat doesn't restart, a Deleted report taken at
// once, a report under an empty root taken (#101 run 2), one item per
// file joined by PresentationUniqueKey, ids derived from the path so a
// moved file is a new item that takes watch data by provider id in either
// order, the recursive refresh of a library shown in its RefreshStatus,
// the RefreshLibrary task, UserData writes answered with success that
// keep the play count and last played date, transcode info only after the
// player reports its playback, LibraryChanged pushes, and X-Emby-Token or
// ?api_key= on every call and socket.
package emby

import (
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/mediabrowser"
	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

const name = "emby"

// Versions are the versions this fake plays.
var Versions = []string{"4.10.1.0"}

var reports, scans, items = mediabrowser.Captures(name)

var (
	// Reports records each path report a front took.
	Reports = reports
	// Scans records each library refresh or full scan a front asked for.
	Scans = scans
	// Items records each item the server added, changed or removed.
	Items = items
)

// Fake is the Emby an Admin adds by address, at every Node's front. Its
// address is typed, so Endpoints sets nothing.
var Fake = fake.Spec{
	Name:      name,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.OnlineSuite("emby"),
	Cues:      mediabrowser.Cues(name),
	Captures:  []fake.CaptureSpec{Reports, Scans, Items},
}

var product = mediabrowser.Product{
	Name: name, Flavor: mediabrowser.Emby, ProductName: "Emby Server",
	TaskName: "Scan media library", TaskID: "6330ee8fb4a957f33981f89aa78b030f",
	Behavior: mediaserver.Behavior{
		MonitorDelay: 90 * time.Second, DeletesAtOnce: true, ItemPerFile: true, NewIDOnPathChange: true,
		Carry: mediaserver.CarryEitherOrder, UserDataIgnoresPlays: true, PurgesVacated: true, ScanTime: time.Second,
	},
}

type (
	Server  = mediabrowser.Server
	User    = mediabrowser.User
	Library = mediabrowser.Library
	Play    = mediabrowser.Play
	Played  = mediabrowser.Played
	Stop    = mediabrowser.Stop
)

// Playing is Fake playing srv, for a test that names it.
func Playing(srv Server) fake.Spec {
	spec := Fake
	spec.New = func(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
		return mediabrowser.Serve(env, product, &srv)
	}
	return spec
}

// newServer plays no Emby until a Scenario names one.
func newServer(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
	return mediabrowser.Serve(env, product, nil)
}
