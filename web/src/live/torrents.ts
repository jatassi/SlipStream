import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { useSyncExternalStore } from "react";

import type { DownloadclientsLiveRow, DownloadclientsLiveTick } from "../api/gen/model";

// Each Torrent's latest torrents.live sample, outside React Query. A tick
// carries only the Torrents that moved, so only their rows re-render. The
// samples belong to the app's QueryClient, so they go with its cache.

export type LiveRow = DownloadclientsLiveRow;

type Store = {
  readonly rows: Map<string, LiveRow>;
  readonly listeners: Map<string, Set<() => void>>;
};

const stores = new WeakMap<QueryClient, Store>();

function storeOf(queryClient: QueryClient): Store {
  const known = stores.get(queryClient);
  if (known !== undefined) {
    return known;
  }
  const store: Store = { rows: new Map(), listeners: new Map() };
  stores.set(queryClient, store);
  return store;
}

function notify({ listeners }: Store, ids: Iterable<string>): void {
  for (const id of ids) {
    for (const listener of listeners.get(id) ?? []) {
      listener();
    }
  }
}

export function takeTorrents(queryClient: QueryClient, tick: DownloadclientsLiveTick): void {
  const store = storeOf(queryClient);
  for (const row of tick.rows) {
    store.rows.set(row.id, row);
  }
  notify(
    store,
    tick.rows.map(({ id }) => id),
  );
}

/** Drops every sample: after a resync the next tick says what still moves. */
export function forgetTorrents(queryClient: QueryClient): void {
  const store = storeOf(queryClient);
  const ids = [...store.rows.keys()];
  store.rows.clear();
  notify(store, ids);
}

function subscribeTo({ listeners }: Store, id: string, listener: () => void): () => void {
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
  const store = storeOf(useQueryClient());
  return useSyncExternalStore(
    (listener) => subscribeTo(store, id, listener),
    () => store.rows.get(id),
  );
}
