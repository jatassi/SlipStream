package health

import (
	"errors"
	"testing"

	"github.com/jatassi/slipstream/internal/setting"
)

func gb(n int64) *setting.Amount { return &setting.Amount{N: n * setting.GB, Unit: setting.BytesUnit} }

func pct(n int64) *setting.Amount { return &setting.Amount{N: n, Unit: setting.PercentUnit} }

func TestResolve(t *testing.T) {
	const tb = setting.TB
	cases := []struct {
		name        string
		global, own Pair
		total       int64
		want        VolumeLevels
	}{
		{
			name:  "the defaults are 100 GB and 20 GB",
			total: tb,
			want:  VolumeLevels{Warn: Resolved{Amount: *gb(100), Bytes: 100 * setting.GB}, Error: Resolved{Amount: *gb(20), Bytes: 20 * setting.GB}},
		},
		{
			name:   "a global percent is a share of the volume's total",
			global: Pair{Warn: pct(10), Error: pct(2)},
			total:  tb,
			want:   VolumeLevels{Warn: Resolved{Amount: *pct(10), Bytes: tb / 10}, Error: Resolved{Amount: *pct(2), Bytes: tb / 50}},
		},
		{
			name:   "a volume's own level wins over the global one",
			global: Pair{Warn: pct(10), Error: pct(2)},
			own:    Pair{Warn: gb(500)},
			total:  tb,
			want:   VolumeLevels{Warn: Resolved{Amount: *gb(500), Bytes: 500 * setting.GB, Own: true}, Error: Resolved{Amount: *pct(2), Bytes: tb / 50}},
		},
		{
			name:  "an own error level leaves the warning on the default",
			own:   Pair{Error: pct(5)},
			total: 2 * tb,
			want:  VolumeLevels{Warn: Resolved{Amount: *gb(100), Bytes: 100 * setting.GB}, Error: Resolved{Amount: *pct(5), Bytes: 2 * tb / 20, Own: true}},
		},
		{
			name:   "a percent of a volume with no total is nothing",
			global: Pair{Warn: pct(10), Error: gb(20)},
			want:   VolumeLevels{Warn: Resolved{Amount: *pct(10)}, Error: Resolved{Amount: *gb(20), Bytes: 20 * setting.GB}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolve(c.global, c.own, c.total); got != c.want {
				t.Fatalf("resolve = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestJudgeIsStrictlyBelowAndErrorFirst(t *testing.T) {
	lv := resolve(Pair{}, Pair{}, setting.TB)
	cases := []struct {
		free int64
		want Level
	}{
		{0, Fail},
		{20*setting.GB - 1, Fail},
		{20 * setting.GB, Warn},
		{100*setting.GB - 1, Warn},
		{100 * setting.GB, None},
		{setting.TB, None},
	}
	for _, c := range cases {
		if got := lv.Judge(c.free); got != c.want {
			t.Errorf("Judge(%d) = %v, want %v", c.free, got, c.want)
		}
	}
	mixed := resolve(Pair{Warn: gb(10), Error: pct(5)}, Pair{}, setting.TB)
	if got := mixed.Judge(40 * setting.GB); got != Fail {
		t.Errorf("below a mixed-unit error level above the warning: Judge = %v, want fail", got)
	}
}

func TestOrdered(t *testing.T) {
	cases := []struct {
		name   string
		levels Pair
		refuse bool
	}{
		{name: "the defaults", levels: Pair{}},
		{name: "an error below the warning", levels: Pair{Warn: gb(50), Error: gb(10)}},
		{name: "an error at the warning", levels: Pair{Warn: gb(10), Error: gb(10)}},
		{name: "zero for both", levels: Pair{Warn: gb(0), Error: gb(0)}},
		{name: "an error above the warning", levels: Pair{Warn: gb(10), Error: gb(11)}, refuse: true},
		{name: "a percent error above a percent warning", levels: Pair{Warn: pct(5), Error: pct(10)}, refuse: true},
		{name: "an error above the default warning", levels: Pair{Error: gb(200)}, refuse: true},
		{name: "mixed units", levels: Pair{Warn: gb(10), Error: pct(50)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ordered(resolve(c.levels, Pair{}, setting.TB))
			refused, ok := errors.AsType[*setting.RangeError](err)
			switch {
			case c.refuse && (!ok || refused.Key != StorageErrorBelow.Key || refused.Sentence == ""):
				t.Fatalf("ordered = %v, want a RangeError on %s", err, StorageErrorBelow.Key)
			case !c.refuse && err != nil:
				t.Fatalf("ordered = %v, want nil", err)
			}
		})
	}
}

func TestSkipsWarning(t *testing.T) {
	cases := []struct {
		name   string
		levels Pair
		total  int64
		want   bool
	}{
		{name: "the defaults", total: setting.TB},
		{name: "a percent error above a bytes warning", levels: Pair{Warn: gb(10), Error: pct(5)}, total: setting.TB, want: true},
		{name: "the same levels on a smaller volume", levels: Pair{Warn: gb(10), Error: pct(5)}, total: 100 * setting.GB},
		{name: "a warning that never fires", levels: Pair{Warn: gb(0), Error: gb(0)}, total: setting.TB},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolve(c.levels, Pair{}, c.total).SkipsWarning(); got != c.want {
				t.Fatalf("SkipsWarning = %v, want %v", got, c.want)
			}
		})
	}
}

func TestDescriptorsCheck(t *testing.T) {
	for _, s := range []setting.Space{StorageWarnBelow, StorageErrorBelow} {
		if err := s.Check(); err != nil {
			t.Error(err)
		}
	}
}
