import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { useSyncExternalStore } from "react";

import type { ConversionsLiveRow, ConversionsLiveSample } from "../api/gen/model";

// Each running Conversion's latest conversions.live sample, outside React
// Query, as live/torrents.ts keeps Torrents': a sample carries only the rows
// that moved, so only those rows re-render.

type Store = {
  readonly rows: Map<string, ConversionsLiveRow>;
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

export function takeConversions(queryClient: QueryClient, sample: ConversionsLiveSample): void {
  const store = storeOf(queryClient);
  for (const row of sample.rows) {
    store.rows.set(row.id, row);
  }
  notify(
    store,
    sample.rows.map(({ id }) => id),
  );
}

/** Drops every sample: after a resync the next one says what still runs. */
export function forgetConversions(queryClient: QueryClient): void {
  const store = storeOf(queryClient);
  const ids = [...store.rows.keys()];
  store.rows.clear();
  notify(store, ids);
}

export function useLiveConversion(id: string): ConversionsLiveRow | undefined {
  const store = storeOf(useQueryClient());
  return useSyncExternalStore(
    (listener) => {
      const own = store.listeners.get(id) ?? new Set();
      own.add(listener);
      store.listeners.set(id, own);
      return () => {
        own.delete(listener);
        if (own.size === 0) {
          store.listeners.delete(id);
        }
      };
    },
    () => store.rows.get(id),
  );
}
