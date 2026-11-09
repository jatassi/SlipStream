package roots_test

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

func (r *storageRig) budgets(t *testing.T) (each map[roots.ID]int, smallest int) {
	t.Helper()
	err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		if each, err = roots.BudgetsIn(ctx, rd); err != nil {
			return err
		}
		smallest, err = roots.SmallestBudget(ctx, rd, alpha)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return each, smallest
}

func TestEachRootsBudgetIsItsNodesPathLimitLessItsFolder(t *testing.T) {
	r := rig(t, false)
	r.run(t)
	other, err := r.svc.Create(t.Context(), identity.Admin{}, roots.NewRoot{ContentType: alpha, Node: role.NodeID{UUID: storage.NewUUID()}, Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	<-r.changed
	mine, theirs := 259-len(r.root.Path)-1, 259-len(other.Path)-1
	if each, smallest := r.budgets(t); each[r.root.ID] != mine || each[other.ID] != theirs || smallest != min(mine, theirs) {
		t.Errorf("before any Node reports the budgets are %v, smallest %d; want %d and %d, smallest %d", each, smallest, mine, theirs, min(mine, theirs))
	}

	f := r.report(storagev1.RootState_ROOT_STATE_READY, r.root.Place)
	f.PathLimit = 4096
	r.arrive(t, f)
	mine = 4096 - len(r.root.Path) - 1
	if each, smallest := r.budgets(t); each[r.root.ID] != mine || each[other.ID] != theirs || smallest != theirs {
		t.Errorf("once one Node reports 4096 the budgets are %v, smallest %d; want %d and %d, smallest %d", each, smallest, mine, theirs, theirs)
	}

	r.storage.mu.Lock()
	delete(r.storage.facts, r.node)
	r.storage.mu.Unlock()
	if each, _ := r.budgets(t); each[r.root.ID] != mine {
		t.Errorf("with its Node offline the root's budget is %d, want the %d it last reported", each[r.root.ID], mine)
	}
}

func TestATypeWithNoRootMeasuresAgainstAnUnreportedNode(t *testing.T) {
	r := rig(t, false)
	var smallest int
	err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		smallest, err = roots.SmallestBudget(ctx, rd, "beta")
		return err
	})
	if err != nil || smallest != 258 {
		t.Errorf("SmallestBudget of a type with no root = %d, %v; want 258", smallest, err)
	}
}
