import { createFileRoute } from "@tanstack/react-router";

import { getGetDefaultGoalSuspenseQueryOptions } from "../../../../api/gen/query/trackers/trackers";
import { ensureQuery } from "../../../../api/query-client";
import { trackerSectionOf } from "../../../../trackers/sections";
import { SeedingTab } from "../../../../trackers/seeding-tab";
import { useTracker } from "../../../../trackers/tracker";
import { Skeleton } from "../../../../ui/skeleton";

const section = trackerSectionOf("seeding");

export const Route = createFileRoute("/_app/settings/trackers/$trackerId/seeding")({
  staticData: { title: section.label },
  loader: ({ context: { queryClient } }) =>
    ensureQuery(queryClient, getGetDefaultGoalSuspenseQueryOptions()),
  pendingComponent: () => <Skeleton label="Loading Seeding" />,
  component: function Seeding() {
    return <SeedingTab tracker={useTracker(Route.useParams().trackerId)} />;
  },
});
