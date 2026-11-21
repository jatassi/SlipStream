import { createFileRoute } from "@tanstack/react-router";

import { PagePending } from "../../../../shell/page";
import { loaderTitle } from "../../../../shell/titles";
import { NodeLayout } from "../-layout";
import { loadNode } from "../-node";

export const Route = createFileRoute("/_app/nodes/$nodeId")({
  staticData: { title: loaderTitle },
  loader: ({ context, params }) => loadNode(context.queryClient, params.nodeId),
  pendingComponent: () => <PagePending label="Loading the Node" shape="facts" />,
  component: function Node() {
    return <NodeLayout nodeId={Route.useParams().nodeId} />;
  },
});
