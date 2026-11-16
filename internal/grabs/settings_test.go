package grabs_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/grabs"
)

func TestEverySearchSettingIsDeclaredSoundly(t *testing.T) {
	for _, d := range []interface{ Check() error }{
		grabs.LongestGap, grabs.FullPace, grabs.StopCount, grabs.StopWindow, grabs.BacklogEvery, grabs.RSSEvery,
	} {
		if err := d.Check(); err != nil {
			t.Error(err)
		}
	}
}
