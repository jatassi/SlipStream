import { RefreshIcon } from "@hugeicons/core-free-icons";
import { type QueryClient, useQueryClient } from "@tanstack/react-query";

import { getListNodesSuspenseQueryOptions } from "../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../api/query-client";
import { askRestart } from "./restart-state";

/** The account menu's, More's and the palette's way to restart the browser's own Node. */
export const restartSelf = { label: "Restart SlipStream…", icon: RefreshIcon } as const;

async function askRestartSelf(queryClient: QueryClient) {
  const { nodes } = await ensureQuery(queryClient, getListNodesSuspenseQueryOptions());
  const self = nodes.find((node) => node.self);
  if (self === undefined) {
    throw new Error("listNodes has no row for the Node that answered");
  }
  askRestart(self);
}

export function useRestartSelf(): () => void {
  const queryClient = useQueryClient();
  return () => {
    void askRestartSelf(queryClient);
  };
}
