import type {
  FilesCell,
  FilesCellList,
  FilesState,
  FilesStateCounts,
  WorksItem,
  WorksWorkView,
} from "../api/gen/model";
import { starter } from "./targets";

const counted = (item: WorksItem) => item.monitored && item.withdrawnAt === null;

function cellOf(item: WorksItem, state: FilesState, edition: string): FilesCell {
  const base = {
    itemId: item.id,
    edition,
    targetId: starter.id,
    targetName: starter.name,
    wanted: counted(item) && (state === "missing" || state === "failed"),
  };
  if (state === "met" || state === "upgradable") {
    return {
      ...base,
      state,
      file: {
        id: item.id.replace(/^.{8}/u, "f11e0000"),
        path: `shelf/${item.key}.epub`,
        size: 1_048_576,
        quality: "webdl-1080p",
        probe: { width: 1920, height: 1080 },
      },
    };
  }
  return { ...base, state };
}

function add(counts: FilesStateCounts, state: FilesState) {
  return { ...counts, [state]: (counts[state] ?? 0) + 1 };
}

/**
 * A many-Item Work's cells, one 1080p cell per Item and Edition that has a
 * state. editions lists each Item's Editions; any unless it says otherwise.
 */
export function shelfCells(
  work: Pick<WorksWorkView, "items">,
  stateOf: (item: WorksItem, index: number) => FilesState | undefined,
  editionsOf: (item: WorksItem, index: number) => readonly string[] = () => ["any"],
): FilesCellList {
  const cells: FilesCell[] = [];
  const items: FilesCellList["items"] = [];
  let summary: FilesCellList["summary"] = { status: null, cells: {}, items: 0, itemStatus: {} };
  work.items.forEach((item, index) => {
    const state = stateOf(item, index);
    items.push({ itemId: item.id, status: state ?? null });
    if (state === undefined) {
      return;
    }
    for (const edition of editionsOf(item, index)) {
      cells.push(cellOf(item, state, edition));
    }
    if (counted(item)) {
      summary = {
        ...summary,
        items: summary.items + 1,
        cells: add(summary.cells, state),
        itemStatus: add(summary.itemStatus, state),
      };
    }
  });
  return { cells, items, summary: { ...summary, status: "missing" }, needsYou: [] };
}
