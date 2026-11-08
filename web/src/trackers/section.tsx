import {
  getGetDefaultGoalSuspenseQueryOptions,
  getGetSeedingSettingsSuspenseQueryOptions,
  getListTrackersSuspenseQueryOptions,
  useListTrackersSuspense,
} from "../api/gen/query/trackers/trackers";
import { getListTasksSuspenseQueryOptions } from "../api/gen/query/tasks/tasks";
import { ensureQuery, freshQuery } from "../api/query-client";
import type { SettingsSection } from "../settings/section";
import { Empty } from "../shell/empty";
import { Section } from "../ui/section";
import { DefaultGoal, RuleSettings } from "./seeding-settings";
import { TrackerList } from "./tracker-list";

function TrackersSection() {
  const { trackers } = useListTrackersSuspense().data;
  return (
    <>
      <DefaultGoal />
      <RuleSettings />
      <Section title="Trackers" count={trackers.length}>
        {trackers.length === 0 ? (
          <Empty>
            No Trackers yet. A Tracker joins this list once an Indexer reaches it or SlipStream
            grabs a Release from it.
          </Empty>
        ) : (
          <TrackerList trackers={trackers} />
        )}
      </Section>
    </>
  );
}

export const trackersSection: SettingsSection = {
  id: "trackers",
  label: "Trackers",
  load: async (queryClient) => {
    await Promise.all([
      ensureQuery(queryClient, getListTrackersSuspenseQueryOptions()),
      freshQuery(queryClient, getGetDefaultGoalSuspenseQueryOptions()),
      freshQuery(queryClient, getGetSeedingSettingsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListTasksSuspenseQueryOptions()),
    ]);
  },
  component: TrackersSection,
  wide: true,
  health: ["seeding"],
};
