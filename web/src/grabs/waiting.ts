import type { QueryClient } from "@tanstack/react-query";

import { getListTorrentsSuspenseQueryOptions } from "../api/gen/query/downloadclients/downloadclients";
import {
  getListGrabsSuspenseQueryOptions,
  getListSearchesSuspenseQueryOptions,
} from "../api/gen/query/grabs/grabs";
import {
  getListIndexersSuspenseQueryOptions,
  useListIndexersSuspense,
} from "../api/gen/query/indexers/indexers";
import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../api/gen/query/nodes/nodes";
import { ensureQuery } from "../api/query-client";
import { waitingLine } from "../nodes/standing";
import { useNow } from "../tasks/use-now";

/** What a Work's searches and Grabs sections read. */
export async function loadWorkGrabs(queryClient: QueryClient, workId: string): Promise<void> {
  await Promise.all([
    ensureQuery(queryClient, getListSearchesSuspenseQueryOptions(workId)),
    ensureQuery(queryClient, getListGrabsSuspenseQueryOptions()),
    ensureQuery(queryClient, getListTorrentsSuspenseQueryOptions()),
    ensureQuery(queryClient, getListIndexersSuspenseQueryOptions()),
    ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
  ]);
}

/** What each Indexer that waits for its home Node says, by Indexer id. */
export function useWaiting(): ReadonlyMap<string, string> {
  const { indexers } = useListIndexersSuspense().data;
  const { nodes } = useListNodesSuspense().data;
  const reading = { now: useNow(60_000), zone: undefined };
  return new Map(
    indexers.flatMap(({ id, waits }) => {
      const home = nodes.find((node) => node.id === waits?.nodeId);
      return waits?.kind !== "waiting" ||
        waits.since === null ||
        home === undefined ||
        home.standing === "in-step"
        ? []
        : [[id, waitingLine(home.name, home.standing, waits.since, reading)] as const];
    }),
  );
}
