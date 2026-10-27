import { useId } from "react";

import type { IndexersIndexer, IndexersTracker, NodesNode } from "../api/gen/model";
import { rowGroup } from "../ui/row-keys";
import { IndexerRow } from "./indexer-row";

export type Group = {
  /** Null gathers the Indexers that reach no Tracker yet. */
  readonly tracker: IndexersTracker | null;
  readonly indexers: readonly IndexersIndexer[];
};

/**
 * The Indexers by the Tracker they reach, by Tracker name: a feed reaching
 * several Trackers shows under each. Those reaching none come last.
 */
export function groupsOf(
  indexers: readonly IndexersIndexer[],
  trackers: readonly IndexersTracker[],
): Group[] {
  const reaching = trackers
    .map((tracker) => ({
      tracker,
      indexers: indexers.filter((indexer) => indexer.trackerIds.includes(tracker.id)),
    }))
    .filter((group) => group.indexers.length > 0);
  const none = indexers.filter((indexer) => indexer.trackerIds.length === 0);
  return none.length === 0 ? reaching : [...reaching, { tracker: null, indexers: none }];
}

function Rows({
  label,
  indexers,
  nodes,
}: {
  label: string;
  indexers: readonly IndexersIndexer[];
  nodes: readonly NodesNode[];
}) {
  return (
    <ul
      aria-label={label}
      {...{ [rowGroup]: "" }}
      className="rounded-card bg-glass ring-1 ring-hairline ring-inset"
    >
      {indexers.map((indexer) => (
        <IndexerRow key={indexer.id} indexer={indexer} nodes={nodes} />
      ))}
    </ul>
  );
}

/** One Tracker and the Indexers that reach it. */
export function TrackerGroup({ group, nodes }: { group: Group; nodes: readonly NodesNode[] }) {
  const titleId = useId();
  const title = group.tracker?.name ?? "No Tracker yet";
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-2">
      <h3
        id={titleId}
        className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch"
      >
        {title}
      </h3>
      {group.tracker === null && (
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          A feed of many sites reaches a Tracker for each site once SlipStream grabs a Release from
          it.
        </p>
      )}
      <Rows
        label={
          group.tracker === null ? "Indexers with no Tracker yet" : `Indexers reaching ${title}`
        }
        indexers={group.indexers}
        nodes={nodes}
      />
    </section>
  );
}

/** Every Indexer in one list, while no Indexer reaches a Tracker. */
export function UngroupedIndexers({
  indexers,
  nodes,
}: {
  indexers: readonly IndexersIndexer[];
  nodes: readonly NodesNode[];
}) {
  return <Rows label="Indexers" indexers={indexers} nodes={nodes} />;
}
