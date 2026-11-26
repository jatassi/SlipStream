// Package emby is fake-emby: one Emby server over the shared Media Server
// core, the same server at every Node's front. It copies Emby 4.10.1.0
// (#71, #101, online run 2026-11-14): its own routes
// (/Library/VirtualFolders/Query, /Items?Path= with a file's or a film
// folder's path, /Users/{user}/Items/{id}/UserData,
// /Users/{user}/PlayedItems/{id}, /Users/{user}/FavoriteItems/{id},
// /embywebsocket), path reports under one 90 s timer that a repeat
// doesn't restart, a Deleted report waiting it out like any other, a
// report under an empty root taken (#101 run 2), one item per file joined
// by PresentationUniqueKey, each listing every version of its group by
// mediasource_ and its item id (recorded run 2026-11-26), ids derived from the path so a moved file is a
// new item that takes watch data by provider id in either order, the
// recursive refresh of a library, the RefreshLibrary task, UserData
// writes that keep the favorite, a mark played that sets no play count or
// last played date, transcode info only after the player reports its
// playback, LibraryChanged pushes, Sessions pushed only when a session
// starts or ends, and X-Emby-Token or ?api_key= on every call and socket.
package emby

import (
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/mediabrowser"
	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/fakes/internal/serverfake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

const name = "emby"

// Versions are the versions this fake plays.
var Versions = []string{"4.10.1.0"}

var reports, scans, items, users = mediabrowser.Captures(name)

var (
	// Reports records each path report a front took.
	Reports = reports
	// Scans records each library refresh or full scan a front asked for.
	Scans = scans
	// Items records each item the server added, changed or removed.
	Items = items
	// Users records every user and its policy after each change.
	Users = users
)

// Fake is the Emby an Admin adds by address, at every Node's front. Its
// address is typed, so Endpoints sets nothing.
var Fake = fake.Spec{
	Name:      name,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.OnlineSuite("emby"),
	Cues:      mediabrowser.Cues(name),
	Captures:  []fake.CaptureSpec{Reports, Scans, Items, Users},
	Entered:   serverfake.Entered(name),
}

var product = mediabrowser.Product{
	Name: name, Flavor: mediabrowser.Emby, ProductName: "Emby Server",
	TaskName: "Scan media library", TaskID: "6330ee8fb4a957f33981f89aa78b030f",
	Behavior: mediaserver.Behavior{
		MonitorDelay: 90 * time.Second, ItemPerFile: true, NewIDOnPathChange: true,
		Carry: mediaserver.CarryEitherOrder, PurgesVacated: true, ScanTime: time.Second,
	},
}

type (
	Server     = mediabrowser.Server
	User       = mediabrowser.User
	UserList   = mediabrowser.UserList
	UserRecord = mediabrowser.UserRecord
	FailNext   = mediabrowser.FailNext
	Library    = mediabrowser.Library
	Play       = mediabrowser.Play
	Played     = mediabrowser.Played
	Stop       = mediabrowser.Stop
)

// Playing is Fake playing srv, for a test that names it.
func Playing(srv Server) fake.Spec {
	spec := Fake
	spec.New = func(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
		return mediabrowser.Serve(env, product, &srv)
	}
	return spec
}

// newServer plays the Emby the Scenario names, or none.
func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	return mediabrowser.ServeScenario(env, product, s, Versions)
}
