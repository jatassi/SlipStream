import { createFileRoute } from "@tanstack/react-router";

import { getListIndexersSuspenseQueryOptions } from "../../../../api/gen/query/indexers/indexers";
import { getListNodesSuspenseQueryOptions } from "../../../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../../../api/query-client";
import { IndexersTab } from "../../../../trackers/indexers-tab";
import { trackerSectionOf } from "../../../../trackers/sections";
import { useTracker } from "../../../../trackers/tracker";
import { Skeleton } from "../../../../ui/skeleton";

const section = trackerSectionOf("indexers");

export const Route = createFileRoute("/_app/settings/trackers/$trackerId/indexers")({
  staticData: { title: section.label },
  loader: ({ context: { queryClient } }) =>
    Promise.all([
      ensureQuery(queryClient, getListIndexersSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
    ]),
  pendingComponent: () => <Skeleton label="Loading Indexers" />,
  component: function Indexers() {
    return <IndexersTab tracker={useTracker(Route.useParams().trackerId)} />;
  },
});
