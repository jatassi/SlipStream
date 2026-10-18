import { createFileRoute } from "@tanstack/react-router";

import { getGetRoleBoardSuspenseQueryOptions } from "../../../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../../../api/query-client";
import { loaderTitle } from "../../../../shell/titles";
import { Skeleton } from "../../../../ui/skeleton";
import { loadNode } from "../-node";
import { NodeProblem } from "../-node-problem";
import { Overview } from "../-overview/overview";

export const Route = createFileRoute("/_app/nodes/$nodeId/")({
  staticData: { title: loaderTitle },
  loader: async ({ context, params }) => {
    const titled = await loadNode(context.queryClient, params.nodeId);
    await ensureQuery(context.queryClient, getGetRoleBoardSuspenseQueryOptions());
    return titled;
  },
  pendingComponent: () => <Skeleton label="Loading the Node" />,
  errorComponent: NodeProblem,
  component: function NodeOverview() {
    return <Overview nodeId={Route.useParams().nodeId} />;
  },
});
