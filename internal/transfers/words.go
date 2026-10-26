package transfers

import (
	"fmt"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
)

// Reason is why a Queued, Waiting or NeedsSpace Transfer doesn't copy,
// with names naming each Node. Any other state has none.
func (v View) Reason(names map[role.NodeID]string) string {
	source, dest := names[v.Source], names[v.DestNode]
	switch v.State {
	case Queued:
		if v.Ahead == 1 {
			return fmt.Sprintf("Queued behind 1 other Transfer from %s.", source)
		}
		return fmt.Sprintf("Queued behind %d other Transfers from %s.", v.Ahead, source)
	case NeedsSpace:
		return fmt.Sprintf("Waiting for %s free on %s (%s).", setting.Amount{N: v.Space.Need, Unit: setting.BytesUnit}.Words(), names[v.Space.Node], v.Space.Volume)
	case Waiting:
		return v.Wait.words(names, source, dest)
	case Copying, Failed:
	}
	return ""
}

func (w Wait) words(names map[role.NodeID]string, source, dest string) string {
	node := names[w.Node]
	switch w.Kind {
	case NodeAway:
		return fmt.Sprintf("Waiting for %s to come back.", node)
	case NodeOtherBuild:
		return fmt.Sprintf("Waiting for %s to update.", node)
	case Connecting:
		return fmt.Sprintf("Connecting to %s.", node)
	case NoRoute:
		return fmt.Sprintf("%s and %s can't reach each other. Open the link port on one of them, or add a Relay Node.", source, dest)
	case RootNotReady:
		return fmt.Sprintf("Waiting for the library root on %s.", node)
	case StorageDraining:
		return fmt.Sprintf("Waiting while %s drains Storage.", node)
	case SourceChecking:
		return fmt.Sprintf("Waiting for %s to check %s.", node, w.Name)
	case SourceNotReady:
		return fmt.Sprintf("Waiting for %s to open %s.", node, w.Name)
	}
	return ""
}

// sentence is a failure's words, with names naming each Node.
func sentence(t Transfer, dest role.NodeID, code Code, position int, detail string, names map[role.NodeID]string) string {
	source := names[t.Source]
	file := "the files"
	if position >= 0 {
		file = t.Files[position].Name
	}
	switch code {
	case SourceGone:
		return fmt.Sprintf("%s no longer has %s.", source, file)
	case SourceFailed:
		if detail == "" {
			return fmt.Sprintf("%s's torrent client failed %s.", source, file)
		}
		return fmt.Sprintf("%s's torrent client failed %s: %s.", source, file, detail)
	case Damaged:
		if detail == readBack {
			return fmt.Sprintf("%s kept reading back damaged on %s. Check %s's disk.", file, names[dest], names[dest])
		}
		return fmt.Sprintf("%s kept arriving damaged from %s. Check %s's memory.", file, source, source)
	case SourceWrittenOff:
		return fmt.Sprintf("%s was written off before %s was copied.", source, file)
	case Rerouted:
		return fmt.Sprintf("The library root is on %s now, which already has the files.", source)
	}
	panic("transfers: a failure of no known code")
}
