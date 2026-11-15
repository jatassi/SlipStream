import { getGetWorkCellsQueryKey } from "../api/gen/query/files/files";
import { getGetWorkSearchingQueryKey, getListWantedQueryKey } from "../api/gen/query/grabs/grabs";
import { getGetWorkQueryKey } from "../api/gen/query/works/works";
import { everyPath } from "./refresh";
import type { LiveEffects } from "./types";

const everyWork = everyPath(/^\/api\/v1\/works\/[^/]+(\/searching|\/cells)?$/u);

/**
 * What the search queue is doing shows on each Work it names and on
 * Wanted, so a queue change reads them again. No Works named means many
 * changed, as a Backlog search's Write does, so every open Work is read again.
 */
export const searchEffects = {
  "grabs.queue-changed": ({ workIds }, { refresh }) => {
    refresh({ queryKey: getListWantedQueryKey() });
    if (workIds.length === 0) {
      refresh(everyWork);
      return;
    }
    for (const workId of workIds) {
      refresh({ queryKey: getGetWorkSearchingQueryKey(workId) });
      refresh({ queryKey: getGetWorkCellsQueryKey(workId) });
      refresh({ queryKey: getGetWorkQueryKey(workId) });
    }
  },
} satisfies Pick<LiveEffects, "grabs.queue-changed">;
