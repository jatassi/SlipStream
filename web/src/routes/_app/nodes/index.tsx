import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import {
  getGetRoleBoardSuspenseQueryOptions,
  getGetTopologySuspenseQueryOptions,
  getListNodesSuspenseQueryOptions,
} from "../../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../../api/query-client";
import { Skeleton } from "../../../ui/skeleton";
import { NodesLanding } from "./-landing/landing";

export const Route = createFileRoute("/_app/nodes/")({
  staticData: { title: "Nodes" },
  validateSearch: z.object({ view: z.enum(["diagram", "roster"]).optional().catch(undefined) }),
  loader: async ({ context }) => {
    await Promise.all([
      ensureQuery(context.queryClient, getGetTopologySuspenseQueryOptions()),
      ensureQuery(context.queryClient, getGetRoleBoardSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListNodesSuspenseQueryOptions()),
    ]);
  },
  pendingComponent: () => <Skeleton label="Loading Nodes" />,
  component: function Nodes() {
    return <NodesLanding view={Route.useSearch().view} />;
  },
});
