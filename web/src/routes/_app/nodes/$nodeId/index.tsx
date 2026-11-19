import { createFileRoute } from "@tanstack/react-router";

import { loaderTitle } from "../../../../shell/titles";
import { Skeleton } from "../../../../ui/skeleton";
import { loadNode } from "../-node";
import { NodeProblem } from "../-node-problem";
import { Overview } from "../-overview/overview";

export const Route = createFileRoute("/_app/nodes/$nodeId/")({
  staticData: { title: loaderTitle },
  loader: ({ context, params }) => loadNode(context.queryClient, params.nodeId),
  pendingComponent: () => <Skeleton label="Loading the Node" />,
  errorComponent: NodeProblem,
  component: function NodeOverview() {
    return <Overview nodeId={Route.useParams().nodeId} />;
  },
});
