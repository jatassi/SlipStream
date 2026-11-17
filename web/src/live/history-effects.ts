import type { InvalidateQueryFilters, QueryClient } from "@tanstack/react-query";

import { spliceNewest, isHistoryPages } from "./history-splice";
import { everyPath, refresh } from "./refresh";
import type { LiveEffects } from "./types";

const everyFeed = everyPath(/^\/api\/v1\/notifications\/feed$/u);
const summary = everyPath(/^\/api\/v1\/history\/summary$/u);
const historyPath = everyPath(/^\/api\/v1\/history$/u);

/** The Work page's ten newest entries: History's own list, not its pages. */
const workSection = {
  predicate: (query: { queryKey: readonly unknown[] }) =>
    historyPath.predicate(query) && !isHistoryPages(query),
};

function readAgain(queryClient: QueryClient, again: (filters: InvalidateQueryFilters) => void) {
  again(everyFeed);
  spliceNewest(queryClient);
  again(workSection);
  again(summary);
}

/**
 * A Clear this browser made, read as history.changed reads it, so History
 * empties once the Clear is accepted, with or without the live stream.
 */
export function adoptHistoryCleared(queryClient: QueryClient): void {
  readAgain(queryClient, (filters) => {
    void refresh(queryClient, filters);
  });
}

/**
 * A sweep deletes bell rows with their entries, so the bell reads again.
 * History's lists take the newest entries, and its sentence counts again.
 */
export const historyEffects = {
  "history.changed": (_data, { queryClient, refresh: again }) => {
    readAgain(queryClient, again);
  },
} satisfies Pick<LiveEffects, "history.changed">;
