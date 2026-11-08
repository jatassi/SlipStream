import { createFileRoute } from "@tanstack/react-router";

import { getGetDefaultGoalSuspenseQueryOptions } from "../../../../api/gen/query/trackers/trackers";
import { ensureQuery } from "../../../../api/query-client";
import { trackerSectionOf } from "../../../../trackers/sections";
import { loadTorrents } from "../../../../torrents/load";
import { SeedingTab } from "../../../../trackers/seeding-tab";
import { useTracker } from "../../../../trackers/tracker";
import { Windows, firstWindowPages } from "../../../../trackers/windows";
import { Skeleton } from "../../../../ui/skeleton";

const section = trackerSectionOf("seeding");

export const Route = createFileRoute("/_app/settings/trackers/$trackerId/seeding")({
  staticData: { title: section.label },
  loader: ({ context: { queryClient }, params: { trackerId } }) =>
    Promise.all([
      ensureQuery(queryClient, getGetDefaultGoalSuspenseQueryOptions()),
      loadTorrents(queryClient),
      ...firstWindowPages(trackerId).map((options) => ensureQuery(queryClient, options)),
    ]),
  pendingComponent: () => <Skeleton label="Loading Seeding" />,
  component: function Seeding() {
    const tracker = useTracker(Route.useParams().trackerId);
    return (
      <div className="max-w-[640px]">
        <SeedingTab tracker={tracker} />
        <Windows trackerId={tracker.id} name={tracker.name} />
      </div>
    );
  },
});
