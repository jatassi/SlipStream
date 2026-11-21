import { createFileRoute } from "@tanstack/react-router";

import { getListNodesSuspenseQueryOptions } from "../../../api/gen/query/nodes/nodes";
import { getGetReconcilingSuspenseQueryOptions } from "../../../api/gen/query/reconciling/reconciling";
import { ensureQuery } from "../../../api/query-client";
import { ReviewList } from "../../../reconciling/review";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/reconciling/torrents")({
  staticData: {
    title: "Review list",
    trail: [{ title: "Reconciling", pathname: "/reconciling" }],
  },
  loader: async ({ context: { queryClient } }) => {
    await Promise.all([
      ensureQuery(queryClient, getGetReconcilingSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
    ]);
  },
  pendingComponent: () => <Skeleton label="Loading the review list" />,
  component: ReviewList,
});
