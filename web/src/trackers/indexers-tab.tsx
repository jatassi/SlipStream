import type { TrackersTrackerItem } from "../api/gen/model";
import { useListIndexersSuspense } from "../api/gen/query/indexers/indexers";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { IndexerRows } from "../indexers/tracker-group";
import { Empty } from "../shell/empty";

/** The Indexers that search a Tracker, each with its switches. */
export function IndexersTab({ tracker }: { tracker: TrackersTrackerItem }) {
  const indexers = useListIndexersSuspense().data.indexers.filter((indexer) =>
    tracker.indexerIds.includes(indexer.id),
  );
  const { nodes } = useListNodesSuspense().data;
  return indexers.length === 0 ? (
    <Empty>
      No Indexer searches {tracker.name}. SlipStream keeps its Grabs and torrents. If another
      Tracker is the same site, merge {tracker.name} into it.
    </Empty>
  ) : (
    <IndexerRows label={`Indexers searching ${tracker.name}`} indexers={indexers} nodes={nodes} />
  );
}
