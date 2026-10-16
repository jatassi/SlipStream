import type { QueryClient } from "@tanstack/react-query";
import { notFound } from "@tanstack/react-router";

import type { NodesLinkState, NodesNode } from "../../../api/gen/model";
import { getListNodesSuspenseQueryOptions } from "../../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../../api/query-client";
import type { Titled } from "../../../shell/titles";

export function nodeOf(nodes: readonly NodesNode[], nodeId: string): NodesNode {
  const node = nodes.find((candidate) => candidate.id === nodeId);
  if (node === undefined) {
    // oxlint-disable-next-line typescript/only-throw-error -- the router renders its own notFound() as the not-found page
    throw notFound();
  }
  return node;
}

export async function loadNode(queryClient: QueryClient, nodeId: string): Promise<Titled> {
  const { nodes } = await ensureQuery(queryClient, getListNodesSuspenseQueryOptions());
  return { title: nodeOf(nodes, nodeId).name };
}

export type NodeState = "online" | "offline" | "mismatched";

export const stateOfLink: { readonly [S in NodesLinkState]: NodeState } = {
  local: "online",
  up: "online",
  down: "offline",
  mismatched: "mismatched",
};
