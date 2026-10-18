import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { notFound } from "@tanstack/react-router";
import { useEffect } from "react";

import type { NodesNode } from "../../../api/gen/model";
import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../../api/query-client";
import { refresh } from "../../../live/refresh";
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

/** A Node's state in a word or three. */
export const stateWords: { readonly [S in NodeState]: string } = {
  online: "online",
  reconnecting: "reconnecting",
  offline: "offline",
  lapsed: "offline past the grace",
  mismatched: "on another version",
  restarting: "restarting",
};

export const stateOf = (node: NodesNode): Exclude<NodeState, "restarting"> =>
  node.standing === "in-step" ? "online" : node.standing;

/**
 * The Workers restarting: each from the accepted request until its link is
 * up again. Once it is, the restart is cleared, so a later outage reads as
 * offline.
 */
export function useRestarting(nodes: readonly NodesNode[]): ReadonlySet<string> {
  const restarts = useRestartState();
  const restarting = new Set(
    nodes.filter((node) => isWorkerRestarting(restarts, node)).map(({ id }) => id),
  );
  const back = nodes
    .filter(({ id }) => restarts.workers.has(id) && !restarting.has(id))
    .map(({ id }) => id)
    .join(" ");
  useEffect(() => {
    for (const nodeId of back.split(" ").filter((id) => id !== "")) {
      restartStore.dispatch({ type: "back", nodeId });
    }
  }, [back]);
  return restarting;
}

export const useNodeState = (node: NodesNode): NodeState =>
  useRestarting([node]).has(node.id) ? "restarting" : stateOf(node);

/** The port of a listen address such as `[::]:8484` or `127.0.0.2:8080`. */
export const portOf = (address: string): number =>
  Number(address.slice(address.lastIndexOf(":") + 1));

const everyNodeQuery = {
  predicate: ({ queryKey: [path] }: { queryKey: readonly unknown[] }) =>
    typeof path === "string" && /^\/api\/v1\/nodes(?:\/|$)/u.test(path),
};

/** Loads every Node query again, after a change to a Node. */
export function useRefreshNodes(): () => Promise<void> {
  const queryClient = useQueryClient();
  return () => refresh(queryClient, everyNodeQuery);
}
