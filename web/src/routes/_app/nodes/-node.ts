import type { QueryClient } from "@tanstack/react-query";
import { notFound } from "@tanstack/react-router";
import { useEffect } from "react";

import type { NodesNode } from "../../../api/gen/model";
import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../../api/query-client";
import type { Away } from "../../../nodes/standing";
import {
  isWorkerRestarting,
  restartStore,
  useRestartState,
} from "../../../shell/restart/restart-state";
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

export type NodeState = "online" | Away | "restarting";

export const stateOf = (node: NodesNode): Exclude<NodeState, "restarting"> =>
  node.standing === "in-step" ? "online" : node.standing;

/**
 * A Worker restarts from the accepted request until its link is up again.
 * Once it is, the restart is cleared, so a later outage reads as offline.
 */
export function useNodeState(node: NodesNode): NodeState {
  const restarts = useRestartState();
  const asked = restarts.workers.has(node.id);
  const restarting = isWorkerRestarting(restarts, node);
  useEffect(() => {
    if (asked && !restarting) {
      restartStore.dispatch({ type: "back", nodeId: node.id });
    }
  }, [asked, restarting, node.id]);
  return restarting ? "restarting" : stateOf(node);
}

/** The port of a listen address such as `[::]:8484` or `127.0.0.2:8080`. */
export const portOf = (address: string): number =>
  Number(address.slice(address.lastIndexOf(":") + 1));
