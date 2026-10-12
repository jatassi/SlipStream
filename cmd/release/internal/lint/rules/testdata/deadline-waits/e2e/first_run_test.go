package e2e

import (
	"context"
	"testing"
	"time"
)

func TestFirstRun(t *testing.T) {
	time.Sleep(time.Second)
	<-time.After(time.Second)
	ctx := context.Background()
	_ = context.TODO()
	_ = t.Context()
	_, _ = ctx, time.Now()
	t.Skip("flaky")
	t.Skipf("flaky, see %s", "https://github.com/jatassi/SlipStream/issues/1")
	t.Skip("flaky: https://github.com/jatassi/SlipStream/issues/42")
	reason := "https://github.com/jatassi/SlipStream/issues/7"
	t.Skip(reason)
	t.SkipNow()
	t.Run("sub", func(st *testing.T) { _ = st.Context() })
}

func BenchmarkLaunch(b *testing.B) {
	_ = b.Context()
	b.Skipf("https://github.com/jatassi/SlipStream/issues/%d", 3)
}

func helper(tb testing.TB) {
	tb.Skip()
}

type notT struct{}

func (notT) Context() context.Context { return nil }
func (notT) Skip()                    {}

func decoy(x notT) {
	_ = x.Context()
	x.Skip()
}
