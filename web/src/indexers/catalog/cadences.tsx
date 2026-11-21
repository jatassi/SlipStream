import { RSSEveryField } from "../../grabs/settings-section";
import { TaskCadence } from "../../tasks/task-cadence";
import { Fact, Facts } from "../../ui/facts";

/** How often the catalog refreshes, and beside it how often RSS sync reads the Indexers' feeds. */
export function Cadences() {
  return (
    <Facts look="sheet">
      <Fact label="Catalog refresh">
        <TaskCadence task="indexer-catalog-refresh" />
      </Fact>
      <Fact label="RSS sync">
        <TaskCadence task="rss-sync" fallback={<RSSEveryField />} />
      </Fact>
    </Facts>
  );
}
