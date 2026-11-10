package downloadclients_test

import (
	"context"
	"slices"
	"testing"
	"time"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

func faultedIn(t *testing.T, lib library) []dc.FaultedTorrent {
	t.Helper()
	var got []dc.FaultedTorrent
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = dc.Faulted(ctx, r)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func faulting(kind dlv1.FaultKind, at time.Time) *dlv1.TorrentReport {
	r := queued()
	r.Fault = &dlv1.Fault{Kind: kind, AtUnixMs: at.UnixMilli()}
	return r
}

func TestFaultedListsReleaseFaultsAndNeverSetupFaults(t *testing.T) {
	l := newListed(t)
	engineFault := faulting(dlv1.FaultKind_FAULT_KIND_ENGINE_ERROR, reportedAt.Add(-3*time.Minute))
	engineFault.Engine, engineFault.Error = dlv1.EngineState_ENGINE_STATE_FAILED, "invalid piece hashes"
	l.add(t, l.seedbox, "Orchard", engineFault)
	l.add(t, l.seedbox, "Harbor", faulting(dlv1.FaultKind_FAULT_KIND_UNREGISTERED, reportedAt.Add(-2*time.Minute)))
	gone := l.add(t, l.seedbox, "Pantry", faulting(dlv1.FaultKind_FAULT_KIND_NO_PAYLOAD, reportedAt.Add(-time.Minute)))
	removeTorrent(t, l.lib, gone, dc.ByAdmin)

	clientError := queued()
	clientError.Hold, clientError.Error = dlv1.Hold_HOLD_CLIENT_ERROR, "Permission denied"
	l.add(t, l.seedbox, "Cellar", clientError)
	diskFull := queued()
	diskFull.Hold = dlv1.Hold_HOLD_DISK_FULL
	l.add(t, l.seedbox, "Shed", diskFull)
	l.add(t, l.seedbox, "Attic", failed())
	l.add(t, l.attic, "Loft", queued())

	type seen struct {
		Name    string
		Kind    dc.FaultKind
		Removed bool
	}
	var got []seen
	for _, f := range faultedIn(t, l.lib) {
		name := names(l, []dc.Row{{ID: f.ID}})[0]
		if f.InfoHash != l.hashes[name] || f.Node != l.seedbox || f.Client != nil {
			t.Errorf("%s = %+v, want its info hash on seedbox's built-in client", name, f)
		}
		got = append(got, seen{name, f.Fault.Kind, f.Removed})
	}
	want := []seen{{"Orchard", dc.FaultEngineError, false}, {"Harbor", dc.FaultUnregistered, false}, {"Pantry", dc.FaultNoPayload, true}}
	if !slices.Equal(got, want) {
		t.Errorf("Faulted = %+v, want the engine and tracker faults, oldest first, and none of the client error, full disk, Node's own failure or away Node", got)
	}
}
