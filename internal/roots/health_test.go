package roots_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
)

func TestARootInTroubleIsAStorageItem(t *testing.T) {
	tests := []struct {
		name    string
		offline bool
		// state is what the Node reports; unspecified reports nothing.
		state storagev1.RootState
		// item is the item's text, unsure the unsure words; both empty
		// for nothing.
		item, unsure string
	}{
		{name: "ready", state: storagev1.RootState_ROOT_STATE_READY},
		{name: "missing", state: storagev1.RootState_ROOT_STATE_MISSING, item: "The folder isn't there. Restore it, or edit the root to where it is now."},
		{name: "unmounted", state: storagev1.RootState_ROOT_STATE_UNMOUNTED, item: "The folder is there but looks unmounted. Plug in or mount the disk."},
		{name: "foreign", state: storagev1.RootState_ROOT_STATE_FOREIGN, item: "The folder holds another library root's marker. Edit the root to a folder of its own."},
		{name: "unwritable", state: storagev1.RootState_ROOT_STATE_UNWRITABLE, item: "SlipStream can't write to the folder. Let SlipStream write to it."},
		{name: "unresponsive", state: storagev1.RootState_ROOT_STATE_UNRESPONSIVE, item: "The folder isn't responding. Check the disk or share it is on."},
		{name: "checking", unsure: "Waiting for Media server to check the folder."},
		{name: "offline", offline: true, state: storagev1.RootState_ROOT_STATE_UNMOUNTED, unsure: "Media server is offline."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := rig(t, tt.offline)
			if tt.state != storagev1.RootState_ROOT_STATE_UNSPECIFIED {
				f := r.report(tt.state, r.root.Place)
				f.Roots[0].Detail = "input/output error"
				r.storage.send(r.node, f)
			}
			a, err := r.svc.HealthSource().Answer(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			expectAnswer(t, a, health.KeyOf("r", r.root.ID.String()), r.root.Path, tt.item, tt.unsure)
		})
	}
}

// expectAnswer checks a holds only key: a failing item saying item, or
// unsure saying unsure, or neither when both are empty.
func expectAnswer(t *testing.T, a health.Answer, key health.Key, path, item, unsure string) {
	t.Helper()
	switch {
	case item != "":
		if len(a.Items) != 1 || len(a.Unsure) != 0 {
			t.Fatalf("answer = %+v, want one item", a)
		}
		it := a.Items[0]
		if it.Key != key || it.Level != health.Fail || it.Subject != "Alphas library root on Media server" || it.Text != item || it.Action.Kind() != health.ActionCheck {
			t.Errorf("item = %+v, want a failing %q on the Alphas library root offering Check now", it, item)
		}
		want := []health.Fact{{Label: "Path", Text: path}, {Label: "State", Text: strings.SplitAfter(item, ".")[0]}, {Label: "Error", Text: "input/output error"}}
		if !slices.Equal(it.Detail, want) {
			t.Errorf("detail = %+v, want the path, state and OS error", it.Detail)
		}
	case unsure != "":
		if len(a.Items) != 0 || len(a.Unsure) != 1 || a.Unsure[0].Key != key || a.Unsure[0].Text != unsure {
			t.Fatalf("answer = %+v, want only %s unsure: %q", a, key, unsure)
		}
	case len(a.Items) != 0 || len(a.Unsure) != 0:
		t.Fatalf("answer = %+v, want nothing", a)
	}
}

func TestTwoRootsOfATypeOnANodeAreNamedByPath(t *testing.T) {
	r := rig(t, false)
	other := r.second(t)
	r.storage.send(r.node, &storagev1.Facts{Roots: []*storagev1.RootFacts{
		{Id: r.root.ID.Bytes(), Place: r.root.Place, State: storagev1.RootState_ROOT_STATE_MISSING},
		{Id: other.ID.Bytes(), Place: other.Place, State: storagev1.RootState_ROOT_STATE_MISSING},
	}})
	a, err := r.svc.HealthSource().Answer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var subjects []string
	for _, it := range a.Items {
		subjects = append(subjects, it.Subject)
	}
	want := []string{"Alphas library root at " + r.root.Path + " on Media server", "Alphas library root at " + other.Path + " on Media server"}
	slices.Sort(subjects)
	if !slices.Equal(subjects, slices.Sorted(slices.Values(want))) {
		t.Fatalf("subjects = %q, want each named by its path %q", subjects, want)
	}
}

func TestHealthJudgesOnlyVolumesHoldingAReadyRoot(t *testing.T) {
	r := rig(t, false)
	r.run(t)
	r.volumesChanged(t, &storagev1.Facts{Roots: []*storagev1.RootFacts{ready(r.root)}, Volumes: []*storagev1.Volume{volume("a", 40, sampled), volume("b", 90, sampled)}}, "a", "b")
	got, err := r.svc.HealthVolumes().Volumes(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := []health.Volume{{Node: r.node, NodeName: "Media server", Mount: "/mnt/a", Name: "/mnt/a", Total: 100, Free: 40, SampledAt: sampled, Live: true}}
	same := func(a, b health.Volume) bool {
		at := a.SampledAt.Equal(b.SampledAt)
		a.SampledAt, b.SampledAt = time.Time{}, time.Time{}
		return at && a == b
	}
	if !slices.EqualFunc(got, want, same) {
		t.Fatalf("HealthVolumes = %+v, want only the ready root's volume %+v", got, want)
	}

	<-r.changed
	r.arrive(t, r.report(storagev1.RootState_ROOT_STATE_UNMOUNTED, r.root.Place))
	if got, err := r.svc.HealthVolumes().Volumes(t.Context()); err != nil || len(got) != 0 {
		t.Fatalf("HealthVolumes with the root unmounted = %+v, %v; want none, not the parent disk", got, err)
	}
}

func TestSamplingAnOfflineNodesVolumesIsUnreachable(t *testing.T) {
	r := rig(t, true)
	if err := r.svc.HealthVolumes().Sample(t.Context(), r.node); !errors.Is(err, health.ErrUnreachable) {
		t.Fatalf("Sample on an offline Node = %v, want health.ErrUnreachable", err)
	}
	unreached, err := r.svc.HealthSource().Check(t.Context(), health.KeyOf("r", r.root.ID.String()))
	if err != nil || len(unreached) != 1 || unreached[0].Words != "Media server is offline." {
		t.Fatalf("Check now on the root of an offline Node = %+v, %v; want Media server is offline.", unreached, err)
	}
}
