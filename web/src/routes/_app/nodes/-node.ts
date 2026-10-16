import type { QueryClient } from "@tanstack/react-query";
import { notFound } from "@tanstack/react-router";

import type { NodesLinkState, NodesNode } from "../../../api/gen/model";
import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../../api/query-client";
import { isWorkerRestarting, useRestartState } from "../../../shell/restart/restart-state";
import type { Titled } from "../../../shell/titles";

export function nodeOf(nodes: readonly NodesNode[], nodeId: string): NodesNode {
  const node = nodes.find((candidate) => candidate.id === nodeId);
  if (node === undefined) {
    // oxlint-disable-next-line typescript/only-throw-error -- the router renders its own notFound() as the not-found page
    throw notFound();
  }
  return node;
}

export const useNode = (nodeId: string): NodesNode =>
  nodeOf(useListNodesSuspense().data.nodes, nodeId);

export async function loadNode(queryClient: QueryClient, nodeId: string): Promise<Titled> {
  const { nodes } = await ensureQuery(queryClient, getListNodesSuspenseQueryOptions());
  return { title: nodeOf(nodes, nodeId).name };
}

export type NodeState = "online" | "offline" | "mismatched" | "restarting";

export const stateOfLink: { readonly [S in NodesLinkState]: Exclude<NodeState, "restarting"> } = {
  local: "online",
  up: "online",
  down: "offline",
  mismatched: "mismatched",
};

/** A Worker restarts from the accepted request until its link is up again. */
export const useNodeState = (node: NodesNode): NodeState =>
  isWorkerRestarting(useRestartState(), node) ? "restarting" : stateOfLink[node.link.state];

/** The port of a listen address such as `[::]:8484` or `127.0.0.2:8080`. */
export const portOf = (address: string): number =>
  Number(address.slice(address.lastIndexOf(":") + 1));
