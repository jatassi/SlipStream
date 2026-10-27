import {
  getGetIndexerSettingsSuspenseQueryOptions,
  getListIndexersSuspenseQueryOptions,
  getListTrackersSuspenseQueryOptions,
  useListIndexersSuspense,
  useListTrackersSuspense,
} from "../api/gen/query/indexers/indexers";
import type { IndexersIndexer } from "../api/gen/model";
import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../api/gen/query/nodes/nodes";
import { ensureQuery, freshQuery } from "../api/query-client";
import { countWords } from "../format";
import type { SettingsSection } from "../settings/section";
import { Empty } from "../shell/empty";
import { Section } from "../ui/section";
import { LongBackoff } from "./long-backoff";
import { TrackerGroup, UngroupedIndexers, groupsOf } from "./tracker-group";

const indexerNoun = { one: "Indexer", many: "Indexers" } as const;

/** What the list comes to, in a sentence: who needs the Admin first, then who searches ask. */
export function summaryOf(indexers: readonly IndexersIndexer[]): string {
  const needing = indexers.filter((indexer) => indexer.on && indexer.fix !== null);
  if (needing.length === 1) {
    return `${needing[0]?.name} needs you. Open it to see what to do.`;
  }
  if (needing.length > 1) {
    return `${countWords(needing.length, indexerNoun)} need you. Open each to see what to do.`;
  }
  const on = indexers.filter((indexer) => indexer.on);
  if (on.length === 0) {
    return "Every Indexer is off, so searches find nothing. Turn one on to search again.";
  }
  const asked = on.filter((indexer) => indexer.waits === null);
  if (asked.length < on.length) {
    return `Searches ask ${asked.length} of the ${on.length} Indexers that are on. The others are waiting, and each row says why.`;
  }
  return on.length === 1
    ? `Searches ask ${on[0]?.name}.`
    : `Searches ask all ${on.length} Indexers that are on.`;
}

function IndexersSection() {
  const { indexers } = useListIndexersSuspense().data;
  const { trackers } = useListTrackersSuspense().data;
  const { nodes } = useListNodesSuspense().data;
  const groups = groupsOf(indexers, trackers);
  return (
    <>
      <Section title="Indexers" count={indexers.length}>
        <div className="flex flex-col gap-5">
          {indexers.length === 0 ? (
            <Empty>No Indexers yet. Every search asks each Indexer added here that is on.</Empty>
          ) : (
            <p className="text-ink-1">{summaryOf(indexers)}</p>
          )}
          {groups.length === 1 && groups[0]?.tracker === null ? (
            <UngroupedIndexers indexers={indexers} nodes={nodes} />
          ) : (
            groups.map((group) => (
              <TrackerGroup
                key={group.tracker?.id ?? "none"}
                group={group}
                trackers={trackers}
                nodes={nodes}
              />
            ))
          )}
        </div>
      </Section>
      <LongBackoff />
    </>
  );
}

export const indexersSection: SettingsSection = {
  id: "indexers",
  label: "Indexers",
  load: async (queryClient) => {
    await Promise.all([
      ensureQuery(queryClient, getListIndexersSuspenseQueryOptions()),
      ensureQuery(queryClient, getListTrackersSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
      freshQuery(queryClient, getGetIndexerSettingsSuspenseQueryOptions()),
    ]);
  },
  component: IndexersSection,
  wide: true,
  health: ["indexers"],
};
