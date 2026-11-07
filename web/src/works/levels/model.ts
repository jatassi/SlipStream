import type {
  FilesCell,
  FilesCellList,
  FilesItemStatus,
  FilesState,
  FilesStateCounts,
  TargetsCut,
  TargetsProfile,
  TargetsTarget,
  TargetsWants,
  WorksContentType,
  WorksItem,
  WorksLevel,
  WorksWorkView,
} from "../../api/gen/model";
import type { SwitchChecked } from "../../ui/switch";
import { codeOf } from "../fields-by-tier";
import { anyCut, cutNameOf } from "../page/matrix-model";
import { type Entry, foldsOf } from "./fold";

export type LevelTarget = { readonly id: string; readonly name: string };

/** What every level of one Work reads alike: its type, its Targets in Profile order and its wanted Editions. */
export type LevelContext = {
  readonly type: WorksContentType;
  readonly targets: readonly LevelTarget[];
  readonly editions: readonly { readonly key: string; readonly name: string }[];
};

/** A wanted Edition in one level. One no Item of the level has a cell of is unseen there. */
export type Column = {
  readonly edition: string;
  readonly name: string;
  readonly seen: boolean;
};

/** An Item's cells of one Edition by Target, or unseen when it has none of a cut others in its level have. */
export type Group =
  | { readonly kind: "cells"; readonly cells: readonly (FilesCell | undefined)[] }
  | { readonly kind: "unseen" };

export type ItemView = {
  readonly item: WorksItem;
  readonly code: string;
  readonly date: string | null;
  readonly status: FilesState | null;
  readonly unreleased: boolean;
  /** Nobody monitors it. */
  readonly quiet: boolean;
  /** Its source no longer lists it, and it holds a file. */
  readonly withdrawn: boolean;
  /** One per column of its level. */
  readonly groups: readonly Group[];
};

export type Tally = { readonly met: number; readonly of: number };

export type LevelView = {
  readonly level: WorksLevel;
  readonly poster: string | undefined;
  readonly columns: readonly Column[];
  /** Cells of the Items that count, by state. */
  readonly counts: FilesStateCounts;
  /** Items that count and are met, of those that count; null while none counts. */
  readonly tally: Tally | null;
  readonly monitor: SwitchChecked;
  /** Something in it that counts is wanted and not met. */
  readonly opens: boolean;
  readonly entries: readonly Entry<ItemView>[];
  /** Withdrawn Items with files, after the live ones. */
  readonly gone: readonly ItemView[];
};

export function levelContext({
  type,
  profile,
  targets,
  wants,
  cuts,
}: {
  type: WorksContentType;
  profile: TargetsProfile;
  targets: readonly TargetsTarget[];
  wants: TargetsWants;
  cuts: readonly TargetsCut[];
}): LevelContext {
  const nameOf = cutNameOf(cuts);
  const editions = wants.editions.kind === "any" ? [anyCut] : wants.editions.cuts;
  return {
    type,
    targets: profile.targets.map(({ targetId }) => ({
      id: targetId,
      name: targets.find((target) => target.id === targetId)?.name ?? targetId,
    })),
    editions: editions.map((key) => ({ key, name: nameOf(key) })),
  };
}

const hasFile = (cell: FilesCell) => cell.state === "met" || cell.state === "upgradable";
const settled = (state: FilesState) => state === "met" || state === "unreleased";

function monitorOf(items: readonly WorksItem[]): SwitchChecked {
  const on = items.filter(({ monitored }) => monitored).length;
  return on === 0 ? false : on === items.length ? true : "mixed";
}

type Read = {
  readonly cells: ReadonlyMap<string, readonly FilesCell[]>;
  readonly statuses: ReadonlyMap<string, FilesItemStatus>;
};

function readOf(cells: FilesCellList): Read {
  const byItem = new Map<string, FilesCell[]>();
  for (const cell of cells.cells) {
    const held = byItem.get(cell.itemId);
    if (held === undefined) {
      byItem.set(cell.itemId, [cell]);
    } else {
      held.push(cell);
    }
  }
  return { cells: byItem, statuses: new Map(cells.items.map((s) => [s.itemId, s])) };
}

const sameInputs = (a: readonly unknown[], b: readonly unknown[]) =>
  a.length === b.length && a.every((value, index) => value === b[index]);

// An unchanged level or Item keeps its view, so its rows don't redraw. The
// Work's and cells' reads keep each unchanged node's object across refetches.
const levelViews = new WeakMap<WorksLevel, { inputs: unknown[]; view: LevelView }>();
const itemViews = new WeakMap<WorksItem, { inputs: unknown[]; view: ItemView }>();

function itemView(
  context: LevelContext,
  columns: readonly Column[],
  item: WorksItem,
  cells: readonly FilesCell[],
  status: FilesItemStatus | undefined,
): ItemView {
  const seen = columns.map(({ seen: s }) => s).join();
  const inputs = [context, seen, item, status, ...cells];
  const known = itemViews.get(item);
  if (known !== undefined && sameInputs(known.inputs, inputs)) {
    return known.view;
  }
  const view: ItemView = {
    item,
    code: codeOf(context.type, item),
    date: item.availability?.date ?? null,
    status: status?.status ?? null,
    unreleased: status?.status === "unreleased",
    quiet: !item.monitored,
    withdrawn: item.withdrawnAt !== null,
    groups: columns.map(({ edition }) => {
      const held = cells.filter((cell) => cell.edition === edition);
      return held.length === 0
        ? { kind: "unseen" }
        : {
            kind: "cells",
            cells: context.targets.map(({ id }) => held.find((cell) => cell.targetId === id)),
          };
    }),
  };
  itemViews.set(item, { inputs, view });
  return view;
}

function countsOf(cells: readonly FilesCell[]): FilesStateCounts {
  const counts: Partial<Record<FilesState, number>> = {};
  for (const { state } of cells) {
    counts[state] = (counts[state] ?? 0) + 1;
  }
  return counts;
}

function posterOf(type: WorksContentType, level: WorksLevel): string | undefined {
  const art = type.artwork.find(
    (kind) => kind.tier === "level" && kind.depth === level.depth && kind.shape === "poster",
  );
  return level.artwork.find(({ kind }) => kind === art?.key)?.url;
}

const tallyOf = (counting: readonly WorksItem[], read: Read): Tally | null =>
  counting.length === 0
    ? null
    : {
        met: counting.filter((item) => read.statuses.get(item.id)?.status === "met").length,
        of: counting.length,
      };

function levelView(
  context: LevelContext,
  level: WorksLevel,
  items: readonly WorksItem[],
  read: Read,
): LevelView | undefined {
  const cellsOf = (item: WorksItem) => read.cells.get(item.id) ?? [];
  const live = items.filter(({ withdrawnAt }) => withdrawnAt === null);
  const gone = items.filter(
    (item) => item.withdrawnAt !== null && cellsOf(item).some((cell) => hasFile(cell)),
  );
  const shown = [...live, ...gone];
  if (shown.length === 0) {
    return undefined;
  }
  const inputs = [
    context,
    ...shown.flatMap((item) => [item, read.statuses.get(item.id), ...cellsOf(item)]),
  ];
  const known = levelViews.get(level);
  if (known !== undefined && sameInputs(known.inputs, inputs)) {
    return known.view;
  }
  const columns = context.editions.map(({ key, name }) => ({
    edition: key,
    name,
    seen: shown.some((item) => cellsOf(item).some((cell) => cell.edition === key)),
  }));
  const views = (of: readonly WorksItem[]) =>
    of.map((item) => itemView(context, columns, item, cellsOf(item), read.statuses.get(item.id)));
  const counting = live.filter(({ monitored }) => monitored);
  const counted = counting.flatMap((item) => cellsOf(item));
  const view: LevelView = {
    level,
    poster: posterOf(context.type, level),
    columns,
    counts: countsOf(counted),
    tally: tallyOf(counting, read),
    monitor: monitorOf(live),
    opens: counted.some(({ state }) => !settled(state)),
    entries: foldsOf(context.type, views(live)),
    gone: views(gone),
  };
  levelViews.set(level, { inputs, view });
  return view;
}

/** An Item's view and its level's, by the Item's id. */
export function viewOf(
  levels: readonly LevelView[],
  itemId: string,
): { view: ItemView; level: LevelView } | undefined {
  for (const level of levels) {
    const views = [
      ...level.entries.flatMap((entry) =>
        entry.kind === "item" ? [entry.item] : entry.fold.items,
      ),
      ...level.gone,
    ];
    const view = views.find(({ item }) => item.id === itemId);
    if (view !== undefined) {
      return { view, level };
    }
  }
  return undefined;
}

const sectionRank = (level: WorksLevel) => (level.withdrawnAt === null ? (level.aside ? 1 : 0) : 2);

/**
 * The Work's levels as sections in tree order, asides after the main run and
 * withdrawn levels that still hold files last. An unchanged level keeps its
 * view from the previous call.
 */
export function levelsOf(
  context: LevelContext,
  work: Pick<WorksWorkView, "levels" | "items">,
  cells: FilesCellList,
): LevelView[] {
  const read = readOf(cells);
  const byLevel = new Map<string, WorksItem[]>();
  for (const item of work.items) {
    if (item.levelId === null) {
      continue;
    }
    const held = byLevel.get(item.levelId);
    if (held === undefined) {
      byLevel.set(item.levelId, [item]);
    } else {
      held.push(item);
    }
  }
  return work.levels
    .filter(({ id }) => byLevel.has(id))
    .toSorted((a, b) => sectionRank(a) - sectionRank(b))
    .flatMap((level) => levelView(context, level, byLevel.get(level.id) ?? [], read) ?? []);
}
