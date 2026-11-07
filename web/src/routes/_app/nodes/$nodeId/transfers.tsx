import { createFileRoute, notFound } from "@tanstack/react-router";

import { getListNodesSuspenseQueryOptions } from "../../../../api/gen/query/nodes/nodes";
import {
  getGetTransferSettingsSuspenseQueryOptions,
  getListTransferSpeedsSuspenseQueryOptions,
  getListTransfersSuspenseQueryOptions,
} from "../../../../api/gen/query/transfers/transfers";
import { ensureQuery } from "../../../../api/query-client";
import { sectionOf } from "../../../../shell/node-sections";
import { NodeTransfers } from "../../../../transfers/node-transfers";
import { Skeleton } from "../../../../ui/skeleton";
import { nodeOf, useNode } from "../-node";
import { NodeProblem } from "../-node-problem";

const section = sectionOf("transfers");

export const Route = createFileRoute("/_app/nodes/$nodeId/transfers")({
  staticData: { title: section.label },
  loader: async ({ context: { queryClient }, params: { nodeId } }) => {
    const { nodes } = await ensureQuery(queryClient, getListNodesSuspenseQueryOptions());
    if (section.shows?.(nodeOf(nodes, nodeId), nodes) === false) {
      // oxlint-disable-next-line typescript/only-throw-error -- the router renders its own notFound() as the not-found page
      throw notFound();
    }
    await Promise.all([
      ensureQuery(queryClient, getListTransfersSuspenseQueryOptions({ nodeId })),
      ensureQuery(queryClient, getListTransferSpeedsSuspenseQueryOptions()),
      ensureQuery(queryClient, getGetTransferSettingsSuspenseQueryOptions(nodeId)),
    ]);
  },
  pendingComponent: () => <Skeleton label="Loading Transfers" />,
  errorComponent: NodeProblem,
  component: function Transfers() {
    return (
      <div className="max-w-[720px]">
        <NodeTransfers node={useNode(Route.useParams().nodeId)} />
      </div>
    );
  },
});
