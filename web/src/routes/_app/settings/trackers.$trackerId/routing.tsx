import { createFileRoute } from "@tanstack/react-router";

import { getListDownloadClientsSuspenseQueryOptions } from "../../../../api/gen/query/downloadclients/downloadclients";
import { getListNodesSuspenseQueryOptions } from "../../../../api/gen/query/nodes/nodes";
import { getListTrackerRoutesSuspenseQueryOptions } from "../../../../api/gen/query/trackers/trackers";
import { ensureQuery } from "../../../../api/query-client";
import { RoutingTab } from "../../../../trackers/routing-tab";
import { trackerSectionOf } from "../../../../trackers/sections";
import { useTracker } from "../../../../trackers/tracker";
import { Skeleton } from "../../../../ui/skeleton";

const section = trackerSectionOf("routing");

export const Route = createFileRoute("/_app/settings/trackers/$trackerId/routing")({
  staticData: { title: section.label },
  loader: ({ context: { queryClient }, params: { trackerId } }) =>
    Promise.all([
      ensureQuery(queryClient, getListTrackerRoutesSuspenseQueryOptions(trackerId)),
      ensureQuery(queryClient, getListDownloadClientsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
    ]),
  pendingComponent: () => <Skeleton label="Loading Routing" />,
  component: function Routing() {
    return <RoutingTab tracker={useTracker(Route.useParams().trackerId)} />;
  },
});
