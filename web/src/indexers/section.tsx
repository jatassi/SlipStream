import {
  getListIndexersSuspenseQueryOptions,
  getListTrackersSuspenseQueryOptions,
  useListIndexersSuspense,
  useListTrackersSuspense,
} from "../api/gen/query/indexers/indexers";
import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../api/gen/query/nodes/nodes";
import { ensureQuery } from "../api/query-client";
import type { SettingsSection } from "../settings/section";
import { Empty } from "../shell/empty";
import { Section } from "../ui/section";
import { AddIndexerButton } from "./add/add-button";
import { SectionTail, loadSectionTail } from "./section-tail";
import { TrackerGroup, UngroupedIndexers, groupsOf } from "./tracker-group";
import { summaryOf } from "./words";

const addIndexer = <AddIndexerButton />;

function IndexersSection() {
  const { indexers } = useListIndexersSuspense().data;
  const { trackers } = useListTrackersSuspense().data;
  const { nodes } = useListNodesSuspense().data;
  const groups = groupsOf(indexers, trackers);
  return (
    <>
      <Section
        title="Indexers"
        count={indexers.length}
        action={indexers.length === 0 ? undefined : addIndexer}
      >
        <div className="flex flex-col gap-5">
          {indexers.length === 0 ? (
            <Empty action={addIndexer}>
              No Indexers yet. Every search asks each Indexer added here that is on.
            </Empty>
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
      <SectionTail />
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
      loadSectionTail(queryClient),
    ]);
  },
  component: IndexersSection,
  wide: true,
  health: ["indexers"],
};
