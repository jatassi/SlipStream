import { createFileRoute, notFound } from "@tanstack/react-router";

import {
  getGetChallengeSolverSuspenseQueryOptions,
  getListIndexersSuspenseQueryOptions,
} from "../../../../api/gen/query/indexers/indexers";
import { getListNodesSuspenseQueryOptions } from "../../../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../../../api/query-client";
import { ChallengeSolver } from "../../../../indexers/solver/challenge-solver";
import { HomedIndexers } from "../../../../indexers/solver/homed-indexers";
import { sectionOf } from "../../../../shell/node-sections";
import { Skeleton } from "../../../../ui/skeleton";
import { nodeOf, stateOf, stateWords, useNode } from "../-node";
import { NodeProblem } from "../-node-problem";

const section = sectionOf("searching");

export const Route = createFileRoute("/_app/nodes/$nodeId/searching")({
  staticData: { title: section.label },
  loader: async ({ context: { queryClient }, params: { nodeId } }) => {
    const { nodes } = await ensureQuery(queryClient, getListNodesSuspenseQueryOptions());
    if (section.shows?.(nodeOf(nodes, nodeId)) === false) {
      // oxlint-disable-next-line typescript/only-throw-error -- the router renders its own notFound() as the not-found page
      throw notFound();
    }
    await Promise.all([
      ensureQuery(queryClient, getGetChallengeSolverSuspenseQueryOptions(nodeId)),
      ensureQuery(queryClient, getListIndexersSuspenseQueryOptions()),
    ]);
  },
  pendingComponent: () => <Skeleton label="Loading Searching" />,
  errorComponent: NodeProblem,
  component: function Searching() {
    const node = useNode(Route.useParams().nodeId);
    return (
      <div className="max-w-[640px]">
        <ChallengeSolver
          node={node}
          away={node.standing === "in-step" ? null : stateWords[stateOf(node)]}
        />
        <HomedIndexers node={node} />
      </div>
    );
  },
});
