import type { QueryClient } from "@tanstack/react-query";

import type { DownloadclientsTorrentList } from "../api/gen/model";
import {
  getListTorrentsQueryKey,
  listTorrents,
} from "../api/gen/query/downloadclients/downloadclients";
import { refresh } from "./refresh";

export function splice(
  list: DownloadclientsTorrentList,
  fresh: DownloadclientsTorrentList,
  ids: readonly string[],
): DownloadclientsTorrentList {
  const named = new Set(ids);
  const now = new Map(fresh.rows.map((row) => [row.id, row]));
  const known = new Set(list.rows.map((row) => row.id));
  const kept = list.rows.flatMap((row) => {
    if (!named.has(row.id)) {
      return [row];
    }
    const next = now.get(row.id);
    return next === undefined ? [] : [next];
  });
  return {
    rows: [...kept, ...fresh.rows.filter((row) => !known.has(row.id))],
    counts: fresh.counts,
  };
}

type Queue = { readonly ids: Set<string>; running: boolean };

const queues = new WeakMap<QueryClient, Queue>();

async function spliceOnce(queryClient: QueryClient, ids: readonly string[]) {
  const queryKey = getListTorrentsQueryKey();
  const state = queryClient.getQueryState<DownloadclientsTorrentList>(queryKey);
  if (state?.data === undefined) {
    return;
  }
  if (state.fetchStatus === "fetching") {
    await refresh(queryClient, { queryKey, exact: true });
    return;
  }
  const fresh = await listTorrents({ ids: [...ids] });
  queryClient.setQueryData<DownloadclientsTorrentList>(
    queryKey,
    (list) => list && splice(list, fresh, ids),
  );
}

/**
 * Splices the named Torrents into the cached list. Splices run one at a
 * time, so an older answer never lands over a newer one, and ids named
 * while one runs are fetched together in the next.
 */
export async function spliceTorrents(
  queryClient: QueryClient,
  ids: readonly string[],
): Promise<void> {
  const queue = queues.get(queryClient) ?? { ids: new Set<string>(), running: false };
  queues.set(queryClient, queue);
  for (const id of ids) {
    queue.ids.add(id);
  }
  if (queue.running) {
    return;
  }
  queue.running = true;
  try {
    while (queue.ids.size > 0) {
      const batch = [...queue.ids];
      queue.ids.clear();
      // oxlint-disable-next-line no-await-in-loop -- each splice lands before the next asks
      await spliceOnce(queryClient, batch);
    }
  } finally {
    queue.running = false;
  }
}
