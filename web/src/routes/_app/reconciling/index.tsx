import { createFileRoute } from "@tanstack/react-router";

import { getListNodesSuspenseQueryOptions } from "../../../api/gen/query/nodes/nodes";
import { getGetReconcilingSuspenseQueryOptions } from "../../../api/gen/query/reconciling/reconciling";
import { ensureQuery } from "../../../api/query-client";
import { ReconcilingPage } from "../../../reconciling/page";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/reconciling/")({
  staticData: { title: "Reconciling" },
  loader: async ({ context: { queryClient } }) => {
    await Promise.all([
      ensureQuery(queryClient, getGetReconcilingSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
    ]);
  },
  pendingComponent: () => <Skeleton label="Loading Reconciling" />,
  component: ReconcilingPage,
});
