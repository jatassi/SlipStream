import type { FilesCell, WorksContentType, WorksWorkView } from "../../api/gen/model";
import { useStartSearch } from "../../grabs/search";
import { openManualSearch } from "../../manual/sheet-state";
import { useGrabNow } from "../page/grab-now";
import { searchedOf } from "../page/searched";
import type { HeaderActions } from "./level-header";
import type { LevelView } from "./model";
import type { useMonitorParts } from "./use-monitored";

/** A level header's actions on one Work: open, monitor, Search, Manual search and Grab now. */
export function useHeaderActions({
  type,
  work,
  cells,
  toggle,
  monitor,
}: {
  type: WorksContentType;
  work: WorksWorkView;
  cells: readonly FilesCell[];
  toggle: (level: LevelView) => void;
  monitor: ReturnType<typeof useMonitorParts>;
}): HeaderActions {
  const search = useStartSearch();
  const grab = useGrabNow(work.id);
  return {
    toggle,
    monitor: (level, on) => {
      monitor({ levelIds: [level.id], itemIds: [], on });
    },
    search: (level) => {
      const inLevel = new Set(
        work.items.filter(({ levelId }) => levelId === level.id).map(({ id }) => id),
      );
      search.start(
        work.id,
        searchedOf(type, work.label, cells, (id) => inLevel.has(id)),
        {
          levelIds: [level.id],
          itemIds: [],
        },
      );
    },
    grabNow: (_, { waits }) => {
      grab.grab(waits);
    },
    manual: (level) => {
      openManualSearch({ workId: work.id, scope: { levelIds: [level.id] } });
    },
  };
}
