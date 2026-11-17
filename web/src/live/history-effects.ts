import { spliceNewest, isHistoryPages } from "./history-splice";
import { everyPath } from "./refresh";
import type { LiveEffects } from "./types";

const everyFeed = everyPath(/^\/api\/v1\/notifications\/feed$/u);
const summary = everyPath(/^\/api\/v1\/history\/summary$/u);
const historyPath = everyPath(/^\/api\/v1\/history$/u);

/** The Work page's ten newest entries: History's own list, not its pages. */
const workSection = {
  predicate: (query: { queryKey: readonly unknown[] }) =>
    historyPath.predicate(query) && !isHistoryPages(query),
};

/**
 * A sweep deletes bell rows with their entries, so the bell reads again.
 * History's lists take the newest entries, and its sentence counts again.
 */
export const historyEffects = {
  "history.changed": (_data, { queryClient, refresh }) => {
    refresh(everyFeed);
    spliceNewest(queryClient);
    refresh(workSection);
    refresh(summary);
  },
} satisfies Pick<LiveEffects, "history.changed">;
