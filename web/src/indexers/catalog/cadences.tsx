import { RSSEveryField } from "../../grabs/settings-section";
import { KeyValues } from "../../routes/_app/nodes/-facts";
import { TaskCadence } from "../../tasks/task-cadence";

/** How often the catalog refreshes, and beside it how often RSS sync reads the Indexers' feeds. */
export function Cadences() {
  return (
    <KeyValues
      rows={[
        ["Catalog refresh", <TaskCadence key="refresh" task="indexer-catalog-refresh" />],
        ["RSS sync", <TaskCadence key="rss" task="rss-sync" fallback={<RSSEveryField />} />],
      ]}
    />
  );
}
