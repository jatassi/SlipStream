package moves

import (
	"slices"

	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storing"
)

// Color is a unit's verdict: green moves, amber moves with a note, red
// stays.
type Color uint8

const (
	Green Color = iota
	Amber
	Red
)

func (c Color) String() string { return [...]string{"green", "amber", "red"}[c] }

// Check is one row of the verdict table.
type Check string

const (
	CheckSpace        Check = "space"
	CheckNames        Check = "names"
	CheckFolder       Check = "folder"
	CheckDestination  Check = "destination"
	CheckMediaServers Check = "media_servers"
	CheckWaits        Check = "waits"
	CheckMoving       Check = "moving"
	CheckProbe        Check = "probe"
	CheckHitAndRun    Check = "hit_and_run"
)

// Reason is one check's answer worse than green. Code is stable per
// reason; Words is its sentence.
type Reason struct {
	Check Check  `json:"check"`
	Color Color  `json:"color"`
	Code  string `json:"code"`
	Words string `json:"words"`
}

type Verdict struct {
	Color   Color
	Reasons []Reason
}

// planFacts is what the planner read for one unit.
type planFacts struct {
	from, to roots.Destination
	// moving is an open unit of another batch moving the Work.
	moving bool
	// probe is nil when every page answered. missing is each planned
	// source file the probe didn't find, occupied each destination path
	// another file holds.
	probe    error
	missing  int
	occupied int
	covers   []mediaservers.Cover
	playing  bool
	// budget is the destination root's path budget, and longest the
	// longest Rel of the unit's files, or of a queued rename's target.
	budget, longest int
	// room is the bytes left on the destination volume before this unit,
	// nil when no copy goes there or its volume is unknown.
	room *int64
}

type check struct {
	Check Check
	run   func(u PlanUnit, f planFacts) (Reason, bool)
}

// checks is the verdict table. Each check is pure and answers its worst
// reason; the unit's color is the worst of all.
var checks = []check{
	{CheckDestination, destinationCheck},
	{CheckMoving, movingCheck},
	{CheckProbe, probeCheck},
	{CheckNames, namesCheck},
	{CheckFolder, folderCheck},
	{CheckSpace, spaceCheck},
	{CheckMediaServers, coverageCheck},
	{CheckWaits, waitsCheck},
}

func verdictOf(u PlanUnit, f planFacts) Verdict {
	var v Verdict
	for _, c := range checks {
		if r, ok := c.run(u, f); ok {
			r.Check = c.Check
			v.Reasons = append(v.Reasons, r)
			v.Color = max(v.Color, r.Color)
		}
	}
	return v
}

var notReady = Reason{Check: CheckDestination, Color: Red, Code: "destination_not_ready", Words: "The destination root isn't ready. Check it in Settings > Library roots."}

func destinationCheck(u PlanUnit, f planFacts) (Reason, bool) {
	switch {
	case u.Subject.From.Root == u.Subject.To.Root:
		return Reason{Color: Red, Code: "same_root", Words: "Its files are already in this root."}, true
	case f.to.ContentType != u.Subject.Type:
		return Reason{Color: Red, Code: "wrong_type", Words: "The destination root holds another kind of title. Pick a root of the same kind."}, true
	case f.to.State != roots.Ready || f.to.Volume == nil:
		return notReady, true
	case f.from.State != roots.Ready || f.from.Volume == nil:
		return Reason{Color: Red, Code: "source_not_ready", Words: "The root its files are in isn't ready. Check it in Settings > Library roots."}, true
	}
	return Reason{}, false
}

func movingCheck(_ PlanUnit, f planFacts) (Reason, bool) {
	if !f.moving {
		return Reason{}, false
	}
	return Reason{Color: Red, Code: "already_moving", Words: "It's already moving in another Move."}, true
}

func probeCheck(u PlanUnit, f planFacts) (Reason, bool) {
	switch {
	case f.probe != nil:
		return Reason{Color: Red, Code: "probe_failed", Words: "SlipStream couldn't read its files on their Node. Preview again once the Node responds."}, true
	case f.missing > 0:
		return Reason{Color: Red, Code: "source_missing", Words: "Some of its files are gone from their root. Scan the library, then preview again."}, true
	case !slices.ContainsFunc(u.Files, func(f File) bool { return f.Kind == KindLibrary }):
		return Reason{Color: Red, Code: "no_files", Words: "It has no files to move."}, true
	}
	return Reason{}, false
}

// namesCheck refuses a unit whose paths don't fit the destination root's
// Node, which opens no path longer than its budget.
func namesCheck(_ PlanUnit, f planFacts) (Reason, bool) {
	if f.longest <= f.budget {
		return Reason{}, false
	}
	return Reason{Color: Red, Code: "names_too_long", Words: "Some of its file paths are too long for the destination root's Node. Shorten its folder and file names, then preview again."}, true
}

func folderCheck(_ PlanUnit, f planFacts) (Reason, bool) {
	if f.occupied == 0 {
		return Reason{}, false
	}
	return Reason{Color: Red, Code: "folder_occupied", Words: "Another file is already where its files go in the destination root. Move or delete that file, then preview again."}, true
}

func spaceCheck(u PlanUnit, f planFacts) (Reason, bool) {
	if f.room == nil || u.Bytes <= *f.room {
		return Reason{}, false
	}
	return Reason{Color: Red, Code: "space_error", Words: "The destination has no room for its files. Free up space there, then preview again."}, true
}

// coverageCheck is red when a server shows the source root and not the
// destination, amber when it shows the destination unconfirmed or never
// scanned.
func coverageCheck(u PlanUnit, f planFacts) (Reason, bool) {
	var worst Reason
	for _, from := range f.covers {
		if from.Root != u.Subject.From.Root || !shows(from.State) {
			continue
		}
		i := slices.IndexFunc(f.covers, func(c mediaservers.Cover) bool { return c.Server == from.Server && c.Root == u.Subject.To.Root })
		switch {
		case i < 0 || !shows(f.covers[i].State):
			return Reason{Color: Red, Code: "mapping_missing", Words: "A Media Server shows its files now but doesn't show the destination root. Add that root to the server's library, then preview again."}, true
		case f.covers[i].State == mediaservers.MapUnconfirmed || f.covers[i].Fresh:
			worst = Reason{Color: Amber, Code: "mapping_unconfirmed", Words: "A Media Server hasn't confirmed it shows the destination root. The server may lose who watched it, their resume positions and ratings."}
		}
	}
	return worst, worst.Color > Green
}

func shows(s mediaservers.MapState) bool {
	return s == mediaservers.MapConfirmed || s == mediaservers.MapUnconfirmed
}

func waitsCheck(_ PlanUnit, f planFacts) (Reason, bool) {
	if !f.playing {
		return Reason{}, false
	}
	return Reason{Color: Amber, Code: "playing", Words: "Someone is playing it now. It switches once they stop."}, true
}

// sameVolume is whether u's source and destination share a volume, so its
// files link and no byte copies.
func sameVolume(from, to roots.Destination) bool {
	return from.Node == to.Node && from.Volume != nil && to.Volume != nil && from.Volume.ID == to.Volume.ID
}

// occupants counts the destination paths another file holds.
func occupants(p storing.Probed, to roots.ID, rels []storing.Rel) int {
	n := 0
	for _, rel := range rels {
		if len(p.Occupants[storing.Library{Root: to.UUID, Rel: rel}]) > 0 {
			n++
		}
	}
	return n
}
