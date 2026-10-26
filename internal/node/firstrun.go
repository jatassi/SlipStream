package node

import "github.com/jatassi/slipstream/internal/firstrun"

// firstRunSteps is every first setup step, one line each, in the order
// firstrun.IDs lists them.
func firstRunSteps(lib *library, h *half) []firstrun.Step {
	return []firstrun.Step{
		lib.ids.AddressStep(),
		lib.ids.PasskeyStep(),
		h.works.MediaStep(),
		h.metadata.LanguageStep(),
		h.roots.RootsStep(),
		h.clients.DownloadsStep(),
		h.targets.TargetsStep(),
	}
}

func (n *Node) openFirstRun(lib *library, h *half) *firstrun.FirstRun {
	return firstrun.New(firstrun.Config{
		Library: lib.db,
		Steps:   firstRunSteps(lib, h),
		Hub:     n.hub,
		Changed: firstrun.ChangedEvent.Bind(n.hub),
		Log:     n.rootLog.With("subsystem", "first run"),
	})
}
