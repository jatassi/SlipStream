import type { InfiniteData, Query, QueryClient } from "@tanstack/react-query";

import type { HistoryListed } from "../api/gen/model";
import { firstPageOf, pagesTag } from "../history/history-query";

type Pages = InfiniteData<HistoryListed, number | undefined>;

export const isHistoryPages = ({ queryKey }: { queryKey: readonly unknown[] }): boolean =>
  queryKey[0] === "/api/v1/history" && queryKey.at(-1) === pagesTag;

/**
 * pages with fresh, a new first page, spliced in: fresh's entries newer than
 * the newest cached one go on top of page 1, and later pages keep their
 * cursors. When fresh no longer holds that entry, a gap or a Clear lies
 * between them, so fresh alone is kept.
 */
export function spliced(pages: Pages, fresh: HistoryListed): Pages {
  const [first, ...rest] = pages.pages;
  const newest = first?.entries[0]?.id;
  if (
    first === undefined ||
    newest === undefined ||
    !fresh.entries.some(({ id }) => id === newest)
  ) {
    return { pages: [fresh], pageParams: [undefined] };
  }
  const added = fresh.entries.filter(({ id }) => id > newest);
  return { ...pages, pages: [{ ...first, entries: [...added, ...first.entries] }, ...rest] };
}

const following = new WeakMap<Query, { again: boolean }>();

/** One read of the first page per burst of events, plus one after it when more arrive meanwhile. */
async function spliceQuery(queryClient: QueryClient, query: Query): Promise<void> {
  const inFlight = following.get(query);
  if (inFlight !== undefined) {
    inFlight.again = true;
    return;
  }
  const slot = { again: false };
  following.set(query, slot);
  try {
    const fresh = await firstPageOf(query.queryHash);
    queryClient.setQueryData<Pages>(query.queryKey, (pages) =>
      pages === undefined ? pages : spliced(pages, fresh),
    );
  } catch {
    // The query's own read reports the failure where the list is drawn.
    await queryClient.invalidateQueries({ predicate: (candidate) => candidate === query });
  } finally {
    following.delete(query);
  }
  if (slot.again) {
    await spliceQuery(queryClient, query);
  }
}

/** Splices the newest entries into each History list on screen, and marks the rest to read again when shown. */
export function spliceNewest(queryClient: QueryClient): void {
  for (const query of queryClient.getQueryCache().findAll({ predicate: isHistoryPages })) {
    if (query.isActive()) {
      void spliceQuery(queryClient, query);
    } else {
      void queryClient.invalidateQueries(
        { predicate: (candidate) => candidate === query },
        { cancelRefetch: false },
      );
    }
  }
}
