import type { QueryClient } from "@tanstack/react-query";

import type { WorksWorkMonitored, WorksWorkView } from "../api/gen/model";
import { getGetWorkQueryKey } from "../api/gen/query/works/works";

/** The Work with the change applied: a level stands for every Item under it, nested levels included. */
function withMonitored(work: WorksWorkView, change: WorksWorkMonitored): WorksWorkView {
  const levels = new Set(change.levelIds);
  for (const level of work.levels) {
    if (level.parentId !== null && levels.has(level.parentId)) {
      levels.add(level.id);
    }
  }
  const items = new Set(change.itemIds);
  const touched = (levelId: string | null, id: string) =>
    items.has(id) || (levelId !== null && levels.has(levelId));
  return {
    ...work,
    items: work.items.map((item) =>
      touched(item.levelId, item.id) && item.monitored !== change.on
        ? { ...item, monitored: change.on }
        : item,
    ),
  };
}

/**
 * A monitoring change the server took, written into the cached Work. The
 * refetch its works.changed starts then finds the same tree, so the Work
 * page redraws only the rows that changed.
 */
export function noteMonitored(
  queryClient: QueryClient,
  workId: string,
  change: WorksWorkMonitored,
) {
  queryClient.setQueryData(getGetWorkQueryKey(workId), (work: WorksWorkView | undefined) =>
    work === undefined ? work : withMonitored(work, change),
  );
}
