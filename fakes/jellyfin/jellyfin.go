// Package jellyfin is fake-jellyfin: one Jellyfin server over the shared
// Media Server core, the same server at every Node's front. It copies
// Jellyfin 12.2.0 and 12.4.0 (#71, #101, recorded run 2026-11-14 filed
// under conformance/jellyfin/recorded/): virtual folders, path reports
// answered 204 and taken after LibraryMonitorDelay (60 s, settable), a
// per-path timer a repeat report restarts, UpdateType ignored and each
// path resolved to the nearest item it holds, so a report under a root
// never scanned with content is ignored until a full scan that read
// content there, a full scan or a folder added dropping pending reports,
// the RefreshLibrary task with State and LastExecutionResult, ids
// derived from the path so a moved file is a new item that takes watch
// data only by provider id and only when removed first, versions in the
// primary item's MediaSources, users, UserData, collections, /Sessions,
// /socket with SessionsStart and ForceKeepAlive, API keys whose create
// answers no body, DeviceId revocation, and 401 to Emby's sign-in forms
// (X-Emby-Token, X-Emby-Authorization, ?api_key=), since 12.x has legacy
// authorization off.
package jellyfin

import (
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/mediabrowser"
	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/fakes/internal/serverfake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

const name = "jellyfin"

// Versions are the versions this fake plays.
var Versions = []string{"12.2.0", "12.4.0"}

var reports, scans, items = mediabrowser.Captures(name)

var (
	// Reports records each path report a front took.
	Reports = reports
	// Scans records each full scan a front asked for.
	Scans = scans
	// Items records each item the server added, changed or removed.
	Items = items
)

// Fake is the Jellyfin an Admin adds by address, at every Node's front.
// Its address is typed, so Endpoints sets nothing.
var Fake = fake.Spec{
	Name:      name,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.OnlineSuite("jellyfin"),
	Cues:      mediabrowser.Cues(name),
	Captures:  []fake.CaptureSpec{Reports, Scans, Items},
	Entered:   serverfake.Entered(name),
}

var product = mediabrowser.Product{
	Name: name, Flavor: mediabrowser.Jellyfin, ProductName: "Jellyfin Server",
	TaskName: "Scan Media Library", TaskID: "7738148ffcd07979c7ceb148e06b3aed",
	Behavior: mediaserver.Behavior{
		MonitorDelay: time.Minute, RestartsOnRepeat: true, ResolvesNearestItem: true, ScanDropsPending: true,
		NewIDOnPathChange: true, Carry: mediaserver.CarryRemovedFirst, PurgesVacated: true, ScanTime: 500 * time.Millisecond,
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

// newServer plays the Jellyfin the Scenario names, or none.
func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	return mediabrowser.ServeScenario(env, product, s, Versions)
}
