package converting

import (
	"errors"
	"testing"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/role"
)

func TestTheBaseTestDecidesWhetherFFmpegRuns(t *testing.T) {
	processor := encode.Pipeline{Path: encode.Processor}
	qsv := encode.Pipeline{Path: encode.Hardware, API: encode.QSV}
	for _, c := range []struct {
		name   string
		proved Proved
		opened bool
		err    error
		want   role.Fit
	}{
		{"a hardware row passed", Proved{Passed: true, Pipeline: qsv}, true, nil, role.Fits},
		{"a hardware row opened its encoder and the processor passed", Proved{Passed: true, Pipeline: processor}, true, nil, role.Fits},
		{"only the processor ran", Proved{Passed: true, Pipeline: processor}, false, nil, role.Weak},
		{"no row passed", Proved{}, true, nil, role.Fails},
		{"the clip couldn't be made", Proved{}, false, errors.New("make the h264_8 clip"), role.Fails},
	} {
		if got := runsOf(c.proved, c.opened, c.err); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
