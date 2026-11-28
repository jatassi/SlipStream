package transfers

import (
	"fmt"
	"slices"

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
		if v.Wait.Kind == FolderNotReady {
			switch v.Destination.(type) {
			case DownloadFolder:
				return fmt.Sprintf("Waiting for the Download Client's folder on %s.", names[v.Wait.Node])
			case Scratch:
				return fmt.Sprintf("Waiting for the scratch folder on %s.", names[v.Wait.Node])
			}
		}
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
		return fmt.Sprintf("%s and %s can't connect to each other. Open the port in link.listen on one of them.", source, dest)
	case RootNotReady:
		return fmt.Sprintf("Waiting for the library root on %s.", node)
	case StorageDraining:
		return fmt.Sprintf("Waiting while %s finishes its Storage work.", node)
	case SourceChecking:
		return fmt.Sprintf("Waiting for %s to check %s.", node, w.Name)
	case SourceNotReady:
		return fmt.Sprintf("Waiting for %s to open %s.", node, w.Name)
	case FolderNotReady:
		return fmt.Sprintf("Waiting for the Backup destination on %s.", node)
	case Reconciling:
		return "Waiting for SlipStream to finish Reconciling."
	}
	return ""
}

// sentence is a failure's words, with names naming each Node.
func sentence(t Transfer, dest role.NodeID, code Code, position int, detail string, names map[role.NodeID]string) string {
	from := names[t.Source]
	file, path := "the files", false
	if at := slices.IndexFunc(t.Files, func(f File) bool { return f.Position == position }); at >= 0 {
		file = t.Files[at].Name
		path = t.Files[at].fromPath()
	}
	switch code {
	case SourceGone:
		return fmt.Sprintf("%s no longer has %s.", from, file)
	case SourceFailed:
		if path {
			return fmt.Sprintf("%s can't read %s: %s.", from, file, detail)
		}
		if detail == "" {
			return fmt.Sprintf("%s's torrent client couldn't read %s.", from, file)
		}
		return fmt.Sprintf("%s's torrent client couldn't read %s: %s.", from, file, detail)
	case Damaged:
		if detail == readBack {
			return fmt.Sprintf("%s was damaged each time %s read it back from disk. Check %s's disk.", file, names[dest], names[dest])
		}
		return fmt.Sprintf("%s kept arriving damaged from %s. Check %s's memory.", file, from, from)
	case SourceWrittenOff:
		return fmt.Sprintf("SlipStream gave up %s's work before %s was copied.", from, file)
	case Rerouted:
		return fmt.Sprintf("The library root moved to %s, which already has these files. Nothing needs copying.", from)
	case FolderFailed:
		if _, ok := t.Destination.(Scratch); ok {
			return fmt.Sprintf("%s couldn't move %s into its scratch folder. Check that SlipStream can write to the folder.", names[dest], file)
		}
		return fmt.Sprintf("%s couldn't write %s to its Backup destination. Check that the folder has room and that SlipStream can write to it.", names[dest], file)
	}
	panic("transfers: a failure of no known code")
}
