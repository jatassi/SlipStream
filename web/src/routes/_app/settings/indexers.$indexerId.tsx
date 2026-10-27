import { createFileRoute } from "@tanstack/react-router";

import {
  getGetIndexerSuspenseQueryOptions,
  getListIndexerQueriesSuspenseQueryOptions,
  getListTrackersSuspenseQueryOptions,
  useGetIndexerSuspense,
  useListIndexerQueriesSuspense,
  useListTrackersSuspense,
} from "../../../api/gen/query/indexers/indexers";
import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../../api/query-client";
import { IndexerPage } from "../../../indexers/indexer-page";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/indexers/$indexerId")({
  staticData: { title: loaderTitle },
  loader: async ({ context: { queryClient }, params: { indexerId } }) => {
    const [indexer] = await Promise.all([
      ensureQuery(queryClient, getGetIndexerSuspenseQueryOptions(indexerId)),
      ensureQuery(queryClient, getListIndexerQueriesSuspenseQueryOptions(indexerId)),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
      ensureQuery(queryClient, getListTrackersSuspenseQueryOptions()),
    ]);
    return {
      title: indexer.name,
      trail: [
        { title: "Settings", pathname: "/settings" },
        { title: "Indexers", pathname: "/settings/indexers" },
      ],
    };
  },
  pendingComponent: () => <Skeleton label="Loading the Indexer" />,
  component: IndexerRoute,
});

function IndexerRoute() {
  const { indexerId } = Route.useParams();
  const indexer = useGetIndexerSuspense(indexerId).data;
  const { queries } = useListIndexerQueriesSuspense(indexerId).data;
  const { nodes } = useListNodesSuspense().data;
  const { trackers } = useListTrackersSuspense().data;
  return (
    <Page>
      <IndexerPage indexer={indexer} queries={queries} nodes={nodes} trackers={trackers} />
    </Page>
  );
}
