import { createFileRoute } from "@tanstack/react-router";

import { getGetNodeConfigSuspenseQueryOptions } from "../../../../api/gen/query/system/system";
import { ensureQuery } from "../../../../api/query-client";
import { sectionOf } from "../../../../shell/node-sections";
import { Skeleton } from "../../../../ui/skeleton";
import { NodeSettings } from "../-config";
import { useNode } from "../-node";
import { NodeProblem } from "../-node-problem";
import { PendingPort } from "../-pending-port";

const section = sectionOf("web");

export const Route = createFileRoute("/_app/nodes/$nodeId/web")({
  staticData: { title: section.label },
  loader: ({ context, params }) =>
    ensureQuery(context.queryClient, getGetNodeConfigSuspenseQueryOptions(params.nodeId)),
  pendingComponent: () => <Skeleton label="Loading the settings" />,
  errorComponent: NodeProblem,
  component: function WebAndApi() {
    const { nodeId } = Route.useParams();
    return (
      <div className="flex max-w-[640px] flex-col gap-4">
        <PendingPort node={useNode(nodeId)} />
        <NodeSettings nodeId={nodeId} section="web" />
      </div>
    );
  },
});
