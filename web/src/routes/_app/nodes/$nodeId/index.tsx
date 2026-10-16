import { createFileRoute } from "@tanstack/react-router";

import { getGetNodeSystemSuspenseQueryOptions } from "../../../../api/gen/query/system/system";
import { ensureQuery } from "../../../../api/query-client";
import { loaderTitle } from "../../../../shell/titles";
import { Skeleton } from "../../../../ui/skeleton";
import { Facts } from "../-facts";
import { loadNode } from "../-node";
import { NodeProblem } from "../-node-problem";
import { Overview } from "../-overview";

export const Route = createFileRoute("/_app/nodes/$nodeId/")({
  staticData: { title: loaderTitle },
  loader: async ({ context, params }) => {
    const titled = await loadNode(context.queryClient, params.nodeId);
    await ensureQuery(context.queryClient, getGetNodeSystemSuspenseQueryOptions(params.nodeId));
    return titled;
  },
  pendingComponent: () => <Skeleton label="Loading the Node" />,
  errorComponent: function OverviewProblem(props) {
    return (
      <Overview nodeId={Route.useParams().nodeId}>
        <NodeProblem {...props} />
      </Overview>
    );
  },
  component: function NodeOverview() {
    const { nodeId } = Route.useParams();
    return (
      <Overview nodeId={nodeId}>
        <Facts nodeId={nodeId} />
      </Overview>
    );
  },
});
