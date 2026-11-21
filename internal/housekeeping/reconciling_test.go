package housekeeping

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/tasks"
)

type keepForever struct{ setting.Panel }

func (keepForever) Keep(context.Context) (time.Duration, error) { return time.Hour, nil }

func TestAPausedHousekeepingStepEndsWithAnOwnedLine(t *testing.T) {
	var pruned []tasks.UnitKey
	step := func(key tasks.UnitKey, while tasks.WhileReconciling) Step {
		return Plain(key, string(key), while, func(context.Context) error { pruned = append(pruned, key); return nil })
	}
	task := Task(Config{
		Keep:        keepForever{},
		Reconciling: func(context.Context) (bool, error) { return true, nil },
		Steps:       []Step{step("history", tasks.Pauses), step("indexer-queries", tasks.Runs)},
	})
	err := task.Run(t.Context(), tasks.Unit{Key: "history"}, &tasks.Progress{})
	skip, ok := errors.AsType[*tasks.SkipError](err)
	if !ok || !skip.Owned || skip.Cause != "Skipped because SlipStream is Reconciling after a restore. The first Housekeeping run after Reconciling finishes deletes them." {
		t.Fatalf("the paused History step ended with %v, want its owned line", err)
	}
	if err := task.Run(t.Context(), tasks.Unit{Key: "indexer-queries"}, &tasks.Progress{}); err != nil {
		t.Fatal(err)
	}
	if len(pruned) != 1 || pruned[0] != "indexer-queries" {
		t.Fatalf("pruned %q while Reconciling, want only indexer-queries", pruned)
	}
}
