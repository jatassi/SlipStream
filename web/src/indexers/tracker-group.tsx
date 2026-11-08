import { Link } from "@tanstack/react-router";
import { type ReactNode, useId } from "react";

import type { IndexersIndexer, TrackersTracker, NodesNode } from "../api/gen/model";
import {
  getListTrackersSuspenseQueryOptions,
  useListTrackersSuspense,
} from "../api/gen/query/trackers/trackers";
import { KeyValues } from "../routes/_app/nodes/-facts";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { rowGroup } from "../ui/row-keys";
import { IndexerRow } from "./indexer-row";
import { TrackerMenu } from "./tracker-menu";

export type Group = {
  /** Null gathers the Indexers that reach no Tracker yet. */
  readonly tracker: TrackersTracker | null;
  readonly indexers: readonly IndexersIndexer[];
};

/**
 * Every Tracker by name with the Indexers it lists: a feed reaching several
 * Trackers shows under each. Indexers no Tracker lists come last.
 */
export function groupsOf(
  indexers: readonly IndexersIndexer[],
  trackers: readonly TrackersTracker[],
): Group[] {
  const reaching = trackers.map((tracker) => ({
    tracker,
    indexers: indexers.filter((indexer) => tracker.indexerIds.includes(indexer.id)),
  }));
  const listed = new Set(trackers.flatMap((tracker) => tracker.indexerIds));
  const none = indexers.filter((indexer) => !listed.has(indexer.id));
  return none.length === 0 ? reaching : [...reaching, { tracker: null, indexers: none }];
}

/** The Trackers the groups read, for the section's loader. */
export const trackersQuery = getListTrackersSuspenseQueryOptions;

/** The Indexers grouped by the Trackers they reach, and every Tracker. */
export function useTrackerGroups(indexers: readonly IndexersIndexer[]) {
  const { trackers } = useListTrackersSuspense().data;
  return { trackers, groups: groupsOf(indexers, trackers) };
}

/** Indexers as rows with their switches. */
export function IndexerRows({
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

const privacyWords = ({ name, privacy }: TrackersTracker): string =>
  privacy === "unknown"
    ? `Its first torrent shows whether ${name} is private.`
    : `${name} is ${privacy}.`;

const unreached =
  "No Indexer reaches this Tracker. Its Grabs and torrents stay with it. If another Tracker is the same site, merge this one into it.";

const lines = (values: readonly string[]): ReactNode =>
  values.length === 0 ? (
    <span>None</span>
  ) : (
    <span className="flex flex-col">
      {values.map((value) => (
        <Data key={value}>{value}</Data>
      ))}
    </span>
  );

function TrackerMachinery({ tracker, reached }: { tracker: TrackersTracker; reached: boolean }) {
  return (
    <Details
      summary={<span className="text-ink-2">{reached ? privacyWords(tracker) : unreached}</span>}
    >
      <div className="mt-2 w-full">
        <KeyValues
          rows={[
            ["Keys", lines(tracker.keys)],
            ["Announce hosts", lines(tracker.hosts)],
          ]}
        />
      </div>
    </Details>
  );
}

const headingLook = "text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch";

function NoTrackerGroup({
  indexers,
  nodes,
}: {
  indexers: readonly IndexersIndexer[];
  nodes: readonly NodesNode[];
}) {
  const titleId = useId();
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-2">
      <h3 id={titleId} className={headingLook}>
        No Tracker yet
      </h3>
      <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        A feed of many sites reaches a Tracker for each site once SlipStream grabs a Release from
        it.
      </p>
      <IndexerRows label="Indexers with no Tracker yet" indexers={indexers} nodes={nodes} />
    </section>
  );
}

/** One Tracker and the Indexers that reach it. */
export function TrackerGroup({
  group,
  trackers,
  nodes,
}: {
  group: Group;
  trackers: readonly TrackersTracker[];
  nodes: readonly NodesNode[];
}) {
  const titleId = useId();
  const { tracker, indexers } = group;
  if (tracker === null) {
    return <NoTrackerGroup indexers={indexers} nodes={nodes} />;
  }
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-1">
      <div className="flex items-center justify-between gap-3">
        <h3 id={titleId} className={headingLook}>
          <Link
            to="/settings/trackers/$trackerId"
            params={{ trackerId: tracker.id }}
            className="rounded-item underline-offset-4 hover:text-ink-1 hover:underline"
          >
            {tracker.name}
          </Link>
        </h3>
        <TrackerMenu
          tracker={tracker}
          others={trackers.filter((other) => other.id !== tracker.id)}
        />
      </div>
      <TrackerMachinery tracker={tracker} reached={indexers.length > 0} />
      {indexers.length > 0 && (
        <div className="mt-1">
          <IndexerRows
            label={`Indexers reaching ${tracker.name}`}
            indexers={indexers}
            nodes={nodes}
          />
        </div>
      )}
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
  return <IndexerRows label="Indexers" indexers={indexers} nodes={nodes} />;
}
