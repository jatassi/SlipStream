import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { useSyncExternalStore } from "react";

import { getGetWorkCellsQueryKey, getListWorkFilesQueryKey } from "../api/gen/query/files/files";
import { getListSearchesQueryKey } from "../api/gen/query/grabs/grabs";
import { getGetWorkWantsQueryKey } from "../api/gen/query/targets/targets";
import { getGetWorkQueryKey } from "../api/gen/query/works/works";

// The Works deleted since the page loaded, outside React Query, so an open
// Work page can leave without mapping a live event itself.

type Store = { readonly removed: Set<string>; readonly listeners: Set<() => void> };

const stores = new WeakMap<QueryClient, Store>();

function storeOf(queryClient: QueryClient): Store {
  const known = stores.get(queryClient);
  if (known !== undefined) {
    return known;
  }
  const store: Store = { removed: new Set(), listeners: new Set() };
  stores.set(queryClient, store);
  return store;
}

/** Marks the Works deleted for any page open on one, and drops every read of theirs. */
export function noteRemoved(queryClient: QueryClient, workIds: readonly string[]): void {
  const store = storeOf(queryClient);
  for (const id of workIds) {
    store.removed.add(id);
  }
  for (const listener of store.listeners) {
    listener();
  }
  for (const id of workIds) {
    for (const queryKey of [
      getGetWorkQueryKey(id),
      getGetWorkCellsQueryKey(id),
      getGetWorkWantsQueryKey(id),
      getListWorkFilesQueryKey(id),
      getListSearchesQueryKey(id),
    ]) {
      queryClient.removeQueries({ queryKey, exact: true });
    }
  }
}

/** Whether the Work was deleted while this page was open. */
export function useRemoved(workId: string): boolean {
  const store = storeOf(useQueryClient());
  return useSyncExternalStore(
    (listener) => {
      store.listeners.add(listener);
      return () => {
        store.listeners.delete(listener);
      };
    },
    () => store.removed.has(workId),
  );
}
