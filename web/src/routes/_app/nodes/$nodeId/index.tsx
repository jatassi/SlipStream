import { createFileRoute } from "@tanstack/react-router";

import { getGetSystemSuspenseQueryOptions } from "../../../../api/gen/query/system/system";
import { ensureQuery } from "../../../../api/query-client";
import { loaderTitle } from "../../../../shell/titles";
import { Skeleton } from "../../../../ui/skeleton";
import { loadNode } from "../-node";
import { Overview } from "../-overview";

export const Route = createFileRoute("/_app/nodes/$nodeId/")({
  staticData: { title: loaderTitle },
  loader: async ({ context, params }) => {
    const [titled] = await Promise.all([
      loadNode(context.queryClient, params.nodeId),
      ensureQuery(context.queryClient, getGetSystemSuspenseQueryOptions()),
    ]);
    return titled;
  },
  pendingComponent: () => <Skeleton label="Loading the Node" />,
  component: function NodeOverview() {
    return <Overview nodeId={Route.useParams().nodeId} />;
  },
});
