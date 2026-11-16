package autosearch_test

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/nodes"
)

// TestRSSSyncReadsTheFeedWhereSearchingRuns runs RSS sync of Fixture
// Site, whose home is seedbox, as seedbox is cut off and comes back. Cut
// off, the unit skips with seedbox offline. Once seedbox lapses home
// covers Searching and reads the feed from the position seedbox's read
// left, and once seedbox settles again the unit goes back to it.
func TestRSSSyncReadsTheFeedWhereSearchingRuns(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launch(t, switches(false, true))
		synctest.Wait()
		expect(t, "the first run", w.ran(rssTask).lines(), []string{"run-now|succeeded", "Fixture Site|seedbox|succeeded||Read 2 new releases. None matched a Work."})

		down := simtest.Partition(t, w.World)
		expect(t, "the run with seedbox cut off", w.ran(rssTask).lines(), []string{"run-now|skipped", "Fixture Site|seedbox|skipped|seedbox is offline|"})
		time.Sleep(time.Until(down.Add(nodes.OfflineGrace.Default + time.Minute)))
		synctest.Wait()
		expect(t, "the run once seedbox lapsed", w.ran(rssTask).lines(), []string{"run-now|succeeded", "Fixture Site|home|succeeded||Read no new releases."})

		w.Path("home", "seedbox").Clear("")
		simtest.Eventually(t, "seedbox's link is up again", func() bool {
			return slices.ContainsFunc(w.API().Nodes("home"), func(r worldtest.Row) bool { return r.Name == "seedbox" && r.Link.State == "up" })
		})
		time.Sleep(node.Defaults(contenttype.Set{}, install.Dev).Timings.Nodes.Settle)
		synctest.Wait()
		expect(t, "the run once seedbox settled", w.ran(rssTask).lines(), []string{"run-now|succeeded", "Fixture Site|seedbox|succeeded||Read no new releases."})
	})
}
