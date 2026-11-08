import type { ReactNode } from "react";

import type { DownloadclientsTorrentList } from "../api/gen/model";
import { useListTorrentsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import type { Row } from "./model";

/** Some of the Torrents, such as one Tracker's, and what shows while none are. */
export type Scope = { readonly keep: (row: Row) => boolean; readonly empty: ReactNode };

function count<K extends string, C>(
  rows: readonly Row[],
  keys: (row: Row) => readonly K[],
  entry: (key: K, torrents: number) => C,
): C[] {
  const counts = new Map<K, number>();
  for (const key of rows.flatMap((row) => keys(row))) {
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
  return [...counts].map(([key, torrents]) => entry(key, torrents));
}

// The scoped rows come from the one list live events splice, so they stay
// live; the server's counts are for every Torrent, so these are counted here.
function scoped(list: DownloadclientsTorrentList, keep: Scope["keep"]): DownloadclientsTorrentList {
  const rows = list.rows.filter((row) => keep(row));
  return {
    rows,
    counts: {
      byState: count(
        rows,
        (row) => [row.state],
        (state, torrents) => ({ state, torrents }),
      ),
      byTab: count(
        rows,
        (row) => row.tabs,
        (tab, torrents) => ({ tab, torrents }),
      ),
      byNode: count(
        rows,
        (row) => [row.nodeId],
        (nodeId, torrents) => ({ nodeId, torrents }),
      ),
    },
  };
}

/** The Torrents a view shows: every one, or those scope keeps. */
export function useScopedTorrents(scope: Scope | undefined): DownloadclientsTorrentList {
  const all = useListTorrentsSuspense().data;
  return scope === undefined ? all : scoped(all, scope.keep);
}
