package tasks

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	rpctestv1 "github.com/jatassi/slipstream/internal/pb/slipstream/rpctest/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/rpctest/v1/rpctestv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

type knows struct{ node role.NodeID }

func (k knows) Key(node role.NodeID) (link.Key, bool) { return link.Key{1}, node == k.node }

func (knows) Serves(role.Name, role.NodeID) bool { return false }

func TestClassify(t *testing.T) {
	seedbox := role.NodeID{UUID: storage.NewUUID()}
	box := peer{name: "seedbox", version: "2.3.0"}
	cluster := rpc.NewCluster(rpc.NewDispatch(nil, link.Key{}, link.Key{}, nil), knows{seedbox})
	_, notSearching := rpctestv1connect.SearchService.On(cluster, seedbox).Search(t.Context(), &rpctestv1.SearchRequest{})
	_, unpaired := rpctestv1connect.PingService.On(cluster, role.NodeID{UUID: storage.NewUUID()}).Ping(t.Context(), &rpctestv1.PingRequest{})
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	for _, tc := range []struct {
		name  string
		ctx   context.Context
		err   error
		place Place
		want  result
	}{
		{"nil", t.Context(), nil, TiedTo(seedbox), result{outcome: Succeeded}},
		{"a canceled run, though the link reads lost", canceled, rpc.Errorf(rpc.CodeCanceled, "context canceled"), TiedTo(seedbox), result{outcome: Canceled}},
		{"disconnected, tied", t.Context(), fmt.Errorf("tidy: %w", link.ErrDisconnected), TiedTo(seedbox),
			result{outcome: Skipped, cause: "seedbox is offline", node: "seedbox", short: "seedbox offline", dueOnReturn: true}},
		{"disconnected, portable", t.Context(), fmt.Errorf("search: %w", link.ErrDisconnected), PortableOn(role.Searching, seedbox),
			result{outcome: Skipped, cause: "seedbox is offline", node: "seedbox", short: "seedbox offline"}},
		{"mismatched, tied", t.Context(), fmt.Errorf("tidy: %w", link.ErrMismatched), TiedTo(seedbox),
			result{outcome: Skipped, cause: "seedbox runs 2.3.0", node: "seedbox", short: "seedbox runs 2.3.0", dueOnReturn: true}},
		{"mismatched, portable", t.Context(), fmt.Errorf("search: %w", link.ErrMismatched), PortableOn(role.Searching, seedbox),
			result{outcome: Skipped, cause: "seedbox runs 2.3.0", node: "seedbox", short: "seedbox runs 2.3.0"}},
		{"not holder, tied", t.Context(), notSearching, TiedTo(seedbox),
			result{outcome: Skipped, cause: "seedbox no longer holds Searching", node: "seedbox", short: "seedbox no longer holds Searching"}},
		{"not holder, portable", t.Context(), notSearching, PortableOn(role.Searching, seedbox),
			result{outcome: Skipped, cause: "seedbox no longer holds Searching", node: "seedbox", short: "seedbox no longer holds Searching"}},
		{"not in the roster", t.Context(), unpaired, TiedTo(seedbox),
			result{outcome: Skipped, cause: "seedbox is no longer paired", node: "seedbox", short: "seedbox is no longer paired"}},
		{"lost, tied", t.Context(), rpc.Errorf(rpc.CodeUnavailable, "session closed"), TiedTo(seedbox),
			result{outcome: Skipped, cause: "lost seedbox mid-run", node: "seedbox", short: "seedbox offline", dueOnReturn: true}},
		{"lost, portable", t.Context(), rpc.Errorf(rpc.CodeUnavailable, "session closed"), PortableOn(role.Searching, seedbox),
			result{outcome: Skipped, cause: "lost seedbox mid-run", node: "seedbox", short: "seedbox offline"}},
		{"a plain error", t.Context(), errors.New("database is locked"), OnController(), result{outcome: Failed, cause: "database is locked"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := classify(tc.ctx, tc.err, tc.place, box); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("classify = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestRollUp(t *testing.T) {
	ok := result{outcome: Succeeded}
	offline := result{outcome: Skipped, cause: "seedbox is offline", node: "seedbox", short: "seedbox offline", dueOnReturn: true}
	nasOffline := result{outcome: Skipped, cause: "nas is offline", node: "nas", short: "nas offline", dueOnReturn: true}
	locked := result{outcome: Failed, cause: "database is locked"}
	full := result{outcome: Failed, cause: "disk is full"}
	for _, tc := range []struct {
		name     string
		results  []result
		canceled bool
		want     [3]string
	}{
		{"cancel requested", []result{ok, ok, {outcome: Canceled}}, true, [3]string{"canceled", "2 of 3 steps", ""}},
		{"every unit succeeded", []result{ok, ok, ok, ok, ok, ok}, false, [3]string{"succeeded", "6 steps", ""}},
		{"one unit succeeded", []result{ok}, false, [3]string{"succeeded", "1 step", ""}},
		{"every unit skipped", []result{offline, nasOffline, offline}, false, [3]string{"skipped", "Waiting for seedbox and nas", ""}},
		{"none succeeded and one failed", []result{offline, locked, full}, false, [3]string{"failed", "2 of 3 steps failed", "database is locked"}},
		{"some succeeded and one skipped", []result{ok, ok, ok, offline, ok, ok}, false, [3]string{"partial", "5 of 6 steps, seedbox offline", ""}},
		{"some succeeded, one failed and one skipped", []result{ok, locked, offline, ok}, false, [3]string{"partial", "2 of 4 steps, 1 failed, seedbox offline", "database is locked"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, summary, cause := rollUp(setting.Noun{One: "step", Many: "steps"}, tc.results, tc.canceled)
			if got := [3]string{string(r), summary, cause}; got != tc.want {
				t.Fatalf("rollUp = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWarn(t *testing.T) {
	units := []unitSince{{label: "Join Codes", at: at("2026-10-15T04:00:00Z")}, {label: "seedbox", at: at("2026-10-12T04:00:00Z")}}
	backup := housekeeping
	backup.Health = "backups"
	for _, tc := range []struct {
		name string
		task Task
		run  latest
		now  string
		want string
	}{
		{"Failed raises one", housekeeping, latest{result: RunFailed, ended: at("2026-10-16T04:12:00Z"), cause: "database is locked"}, "2026-10-16T05:00:00Z",
			"scheduled-tasks|Housekeeping|failed|2026-10-16T04:12:00Z|database is locked||Housekeeping failed at 04:12: database is locked"},
		{"the next Succeeded clears it", housekeeping, latest{result: RunSucceeded, ended: at("2026-10-17T04:01:00Z")}, "2026-10-17T05:00:00Z", ""},
		{"Partial within the interval multiple raises none", housekeeping, latest{result: RunPartial, ended: at("2026-10-15T03:59:00Z")}, "2026-10-15T03:59:00Z", ""},
		{"Partial past it raises one naming the unit", housekeeping, latest{result: RunPartial, ended: at("2026-10-15T04:01:00Z"), cause: "disk is full"}, "2026-10-15T04:01:00Z",
			"scheduled-tasks|Housekeeping|overdue|2026-10-12T04:00:00Z|disk is full|seedbox|Housekeeping has not finished seedbox since Oct 12 04:00"},
		{"Skipped raises nothing", housekeeping, latest{result: RunSkipped}, "2026-10-20T00:00:00Z", ""},
		{"Canceled raises nothing", housekeeping, latest{result: RunCanceled}, "2026-10-20T00:00:00Z", ""},
		{"a task's own Health is its category", backup, latest{result: RunFailed, ended: at("2026-10-16T04:12:00Z"), cause: "no space"}, "2026-10-16T05:00:00Z",
			"backups|Housekeeping|failed|2026-10-16T04:12:00Z|no space||Housekeeping failed at 04:12: no space"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, ok := warn(tc.task, tc.run, units, 3*day, at(tc.now), time.UTC)
			got := ""
			if ok {
				kind := "overdue"
				if w.Failed {
					kind = "failed"
				}
				got = strings.Join([]string{w.Category, w.Name, kind, w.At.UTC().Format(time.RFC3339), w.Cause, w.Unit, w.Text}, "|")
			}
			if got != tc.want {
				t.Fatalf("warn = %q, want %q", got, tc.want)
			}
		})
	}
}
