package sim_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/internal/mediafix"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/node"
)

// TestScenarioFoldersProbeAsTheirSpec writes home's library folders, which
// no root names, and a Node probes each file as the process world's encode
// of it reads.
func TestScenarioFoldersProbeAsTheirSpec(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		probers := map[string]media.Prober{}
		s := simtest.Home(t)
		w := sim.Launch(t, s, sim.WithWalk(), sim.Edit(func(name string, e *node.Env) { probers[name] = e.Media }))
		dir := w.Manifest().MediaDir
		for _, f := range s.Library.Folders {
			for _, m := range f.Files {
				file := filepath.Join(dir, f.Folder, filepath.FromSlash(m.Path))
				if _, err := os.Stat(file); err != nil {
					t.Fatal(err)
				}
				got, err := probers["home"].Probe(t.Context(), file)
				if want := mediafix.Want(m); err != nil || !reflect.DeepEqual(got, want) {
					t.Errorf("home probes %s as %+v, %v; want %+v", m.Path, got, err, want)
				}
			}
		}
	})
}
