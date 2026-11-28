import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { useSyncExternalStore } from "react";

import type {
  ConversionsLiveRow,
  ConversionsLiveSample,
  ConversionsNodeConverting,
} from "../api/gen/model";
import { getGetNodeConvertingQueryKey } from "../api/gen/query/conversions/conversions";

// Each running Conversion's latest conversions.live sample, outside React
// Query, as live/torrents.ts keeps Torrents': a sample carries only the rows
// that moved, so only those rows re-render. Each Node's running count is
// kept beside them for the Diagram.

type Store = {
  readonly rows: Map<string, ConversionsLiveRow>;
  readonly listeners: Map<string, Set<() => void>>;
  running: ReadonlyMap<string, number>;
  readonly runningListeners: Set<() => void>;
};

const stores = new WeakMap<QueryClient, Store>();

function storeOf(queryClient: QueryClient): Store {
  const known = stores.get(queryClient);
  if (known !== undefined) {
    return known;
  }
  const store: Store = {
    rows: new Map(),
    listeners: new Map(),
    running: new Map(),
    runningListeners: new Set(),
  };
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
  if (sample.nodes.some(({ id, running }) => store.running.get(id) !== running)) {
    const running = new Map(store.running);
    for (const node of sample.nodes) {
      running.set(node.id, node.running);
    }
    setRunning(store, running);
  }
}

function setRunning(store: Store, running: ReadonlyMap<string, number>): void {
  store.running = running;
  for (const listener of store.runningListeners) {
    listener();
  }
}

/** Drops every sample: after a resync the next one says what still runs. */
export function forgetConversions(queryClient: QueryClient): void {
  const store = storeOf(queryClient);
  const ids = [...store.rows.keys()];
  store.rows.clear();
  notify(store, ids);
  setRunning(store, new Map());
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

/** How many Conversions each Node runs, by Node id, as the latest samples said. */
export function useRunningConversions(): ReadonlyMap<string, number> {
  const store = storeOf(useQueryClient());
  return useSyncExternalStore(
    (listener) => {
      store.runningListeners.add(listener);
      return () => {
        store.runningListeners.delete(listener);
      };
    },
    () => store.running,
  );
}

/** Takes a Node's Converting section as a save answered it. */
export function adoptNodeConverting(
  queryClient: QueryClient,
  nodeId: string,
  converting: ConversionsNodeConverting,
): void {
  queryClient.setQueryData(getGetNodeConvertingQueryKey(nodeId), converting);
}
