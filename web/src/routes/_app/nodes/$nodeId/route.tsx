import { createFileRoute } from "@tanstack/react-router";

import { loaderTitle } from "../../../../shell/titles";
import { Skeleton } from "../../../../ui/skeleton";
import { NodeLayout } from "../-layout";
import { loadNode } from "../-node";

export const Route = createFileRoute("/_app/nodes/$nodeId")({
  staticData: { title: loaderTitle },
  loader: ({ context, params }) => loadNode(context.queryClient, params.nodeId),
  pendingComponent: () => <Skeleton label="Loading the Node" />,
  component: function Node() {
    return <NodeLayout nodeId={Route.useParams().nodeId} />;
  },
});
