import { createFileRoute, notFound } from "@tanstack/react-router";

import {
  getListMediaServerKindsSuspenseQueryOptions,
  getListMediaServersSuspenseQueryOptions,
} from "../../../../api/gen/query/mediaservers/mediaservers";
import { getListNodesSuspenseQueryOptions } from "../../../../api/gen/query/nodes/nodes";
import { ensureQuery, freshQuery } from "../../../../api/query-client";
import { NodeServers } from "../../../../media-servers/node-servers";
import { sectionOf } from "../../../../shell/node-sections";
import { Skeleton } from "../../../../ui/skeleton";
import { nodeOf, useNode } from "../-node";
import { NodeProblem } from "../-node-problem";

const section = sectionOf("media-servers");

export const Route = createFileRoute("/_app/nodes/$nodeId/media-servers")({
  staticData: { title: section.label },
  loader: async ({ context: { queryClient }, params: { nodeId } }) => {
    const { nodes } = await ensureQuery(queryClient, getListNodesSuspenseQueryOptions());
    if (section.shows?.(nodeOf(nodes, nodeId), nodes) === false) {
      // oxlint-disable-next-line typescript/only-throw-error -- the router renders its own notFound() as the not-found page
      throw notFound();
    }
    await Promise.all([
      freshQuery(queryClient, getListMediaServersSuspenseQueryOptions()),
      ensureQuery(queryClient, getListMediaServerKindsSuspenseQueryOptions()),
    ]);
  },
  pendingComponent: () => <Skeleton label="Loading Media Servers" />,
  errorComponent: NodeProblem,
  component: function MediaServers() {
    return (
      <div className="max-w-[720px]">
        <NodeServers node={useNode(Route.useParams().nodeId)} />
      </div>
    );
  },
});
