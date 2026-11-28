import { createFileRoute, notFound } from "@tanstack/react-router";

import { getGetNodeConvertingSuspenseQueryOptions } from "../../../../api/gen/query/conversions/conversions";
import { getListNodesSuspenseQueryOptions } from "../../../../api/gen/query/nodes/nodes";
import { getGetNodeConfigSuspenseQueryOptions } from "../../../../api/gen/query/system/system";
import { getListTasksSuspenseQueryOptions } from "../../../../api/gen/query/tasks/tasks";
import { ensureQuery, freshQuery } from "../../../../api/query-client";
import { ConvertingSection } from "../../../../nodes/converting-section";
import { sectionOf } from "../../../../shell/node-sections";
import { Skeleton } from "../../../../ui/skeleton";
import { nodeOf, useNode } from "../-node";
// oxlint-disable-next-line import/max-dependencies -- the loader reads the section's, the Maintenance window's and the Node config's queries
import { NodeProblem } from "../-node-problem";

const section = sectionOf("converting");

export const Route = createFileRoute("/_app/nodes/$nodeId/converting")({
  staticData: { title: section.label },
  loader: async ({ context: { queryClient }, params: { nodeId } }) => {
    const { nodes } = await ensureQuery(queryClient, getListNodesSuspenseQueryOptions());
    const node = nodeOf(nodes, nodeId);
    if (section.shows?.(node, nodes) === false) {
      // oxlint-disable-next-line typescript/only-throw-error -- the router renders its own notFound() as the not-found page
      throw notFound();
    }
    await Promise.all([
      freshQuery(queryClient, getGetNodeConvertingSuspenseQueryOptions(nodeId)),
      ensureQuery(queryClient, getListTasksSuspenseQueryOptions()),
      node.standing === "in-step" &&
        ensureQuery(queryClient, getGetNodeConfigSuspenseQueryOptions(nodeId)),
    ]);
  },
  pendingComponent: () => <Skeleton label="Loading Converting" shape="form" />,
  errorComponent: NodeProblem,
  component: function Converting() {
    return (
      <div className="max-w-[640px]">
        <ConvertingSection node={useNode(Route.useParams().nodeId)} />
      </div>
    );
  },
});
