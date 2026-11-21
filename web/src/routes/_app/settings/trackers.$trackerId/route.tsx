import { createFileRoute } from "@tanstack/react-router";

import {
  getGetTrackerSuspenseQueryOptions,
  getListTrackersSuspenseQueryOptions,
} from "../../../../api/gen/query/trackers/trackers";
import { ensureQuery } from "../../../../api/query-client";
import { PagePending } from "../../../../shell/page";
import { loaderTitle } from "../../../../shell/titles";
import { TrackerLayout } from "../../../../trackers/layout";
import { trackerOf } from "../../../../trackers/tracker";

export const Route = createFileRoute("/_app/settings/trackers/$trackerId")({
  staticData: { title: loaderTitle },
  loader: async ({ context: { queryClient }, params: { trackerId } }) => {
    const [{ trackers }] = await Promise.all([
      ensureQuery(queryClient, getListTrackersSuspenseQueryOptions()),
      ensureQuery(queryClient, getGetTrackerSuspenseQueryOptions(trackerId)),
    ]);
    return {
      title: trackerOf(trackers, trackerId).name,
      trail: [
        { title: "Settings", pathname: "/settings" },
        { title: "Trackers", pathname: "/settings/trackers" },
      ],
    };
  },
  pendingComponent: () => <PagePending label="Loading the Tracker" shape="facts" />,
  component: function Tracker() {
    return <TrackerLayout trackerId={Route.useParams().trackerId} />;
  },
});
