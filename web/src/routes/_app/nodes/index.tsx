import { createFileRoute, redirect } from "@tanstack/react-router";

import { getListNodesSuspenseQueryOptions } from "../../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../../api/query-client";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/nodes/")({
  loaderDeps: ({ search }) => search,
  loader: async ({ context, deps }) => {
    const { nodes } = await ensureQuery(context.queryClient, getListNodesSuspenseQueryOptions());
    const self = nodes.find((node) => node.self);
    if (self === undefined) {
      throw new Error("listNodes has no row for the Node that answered");
    }
    redirect({
      to: "/nodes/$nodeId",
      params: { nodeId: self.id },
      search: deps,
      replace: true,
      throw: true,
    });
  },
  pendingComponent: () => <Skeleton label="Loading Nodes" />,
});
