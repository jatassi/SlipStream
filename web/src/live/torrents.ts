import { useSyncExternalStore } from "react";

import type { DownloadclientsLiveRow, DownloadclientsLiveTick } from "../api/gen/model";

// Each Torrent's latest torrents.live sample, outside React Query. A tick
// carries only the Torrents that moved, so only their rows re-render.

export type LiveRow = DownloadclientsLiveRow;

const rows = new Map<string, LiveRow>();
const listeners = new Map<string, Set<() => void>>();

export function takeTorrents(tick: DownloadclientsLiveTick): void {
  for (const row of tick.rows) {
    rows.set(row.id, row);
  }
  for (const { id } of tick.rows) {
    for (const listener of listeners.get(id) ?? []) {
      listener();
    }
  }
}

function subscribeTo(id: string, listener: () => void): () => void {
  const own = listeners.get(id) ?? new Set();
  own.add(listener);
  listeners.set(id, own);
  return () => {
    own.delete(listener);
    if (own.size === 0) {
      listeners.delete(id);
    }
  };
}

export function useLiveRow(id: string): LiveRow | undefined {
  return useSyncExternalStore(
    (listener) => subscribeTo(id, listener),
    () => rows.get(id),
  );
}
