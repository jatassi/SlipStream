import { createFileRoute } from "@tanstack/react-router";

import { getListNodesSuspenseQueryOptions } from "../../../api/gen/query/nodes/nodes";
import { getGetReconcilingSuspenseQueryOptions } from "../../../api/gen/query/reconciling/reconciling";
import { getListRootsSuspenseQueryOptions } from "../../../api/gen/query/roots/roots";
import { getListContentTypesSuspenseQueryOptions } from "../../../api/gen/query/works/works";
import { ensureQuery, freshQuery } from "../../../api/query-client";
import { FoldersStep } from "../../../reconciling/folders";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/reconciling/folders")({
  staticData: {
    title: "Find the old Controller's folders",
    trail: [{ title: "Reconciling", pathname: "/reconciling" }],
  },
  loader: async ({ context: { queryClient } }) => {
    await Promise.all([
      freshQuery(queryClient, getGetReconcilingSuspenseQueryOptions()),
      ensureQuery(queryClient, getListRootsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
      ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
    ]);
  },
  pendingComponent: () => <Skeleton label="Loading the old Controller's folders" />,
  component: FoldersStep,
});
