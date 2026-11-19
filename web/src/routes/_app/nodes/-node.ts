import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { notFound } from "@tanstack/react-router";

import type { NodesNode } from "../../../api/gen/model";
import {
  getGetRoleBoardSuspenseQueryOptions,
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../../api/query-client";
import { refresh } from "../../../live/refresh";
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

/** Loads the Node and the role board its header's Role chips read. */
export async function loadNode(queryClient: QueryClient, nodeId: string): Promise<Titled> {
  const [{ nodes }] = await Promise.all([
    ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
    ensureQuery(queryClient, getGetRoleBoardSuspenseQueryOptions()),
  ]);
  return { title: nodeOf(nodes, nodeId).name };
}

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
