import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import {
  getGetRoleBoardSuspenseQueryOptions,
  getGetTopologySuspenseQueryOptions,
  getListNodesSuspenseQueryOptions,
} from "../../../api/gen/query/nodes/nodes";
import { getGetRemoteAccessSuspenseQueryOptions } from "../../../api/gen/query/reachability/reachability";
import { ensureQuery } from "../../../api/query-client";
import { PagePending } from "../../../shell/page";
import { NodesLanding } from "./-landing/landing";

export const Route = createFileRoute("/_app/nodes/")({
  staticData: { title: "Nodes" },
  validateSearch: z.object({ view: z.enum(["diagram", "roster"]).optional().catch(undefined) }),
  loader: async ({ context }) => {
    await Promise.all([
      ensureQuery(context.queryClient, getGetTopologySuspenseQueryOptions()),
      ensureQuery(context.queryClient, getGetRoleBoardSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListNodesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getGetRemoteAccessSuspenseQueryOptions()),
    ]);
  },
  pendingComponent: () => <PagePending label="Loading Nodes" shape="rows" lines={2} />,
  component: function Nodes() {
    return <NodesLanding view={Route.useSearch().view} />;
  },
});
