import type {
  FilesCell,
  FilesCellList,
  FilesItemStatus,
  FilesLevelRollup,
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
import { stateOrder } from "../../ui/state-marks";
import { codeOf } from "../fields-by-tier";
import { anyCut, cutNameOf, fileOf } from "../page/matrix-model";
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
  /** The level's id, or the trailing section's. */
  readonly id: string;
  readonly name: string;
  /** The trailing section of the Items whose levels their source no longer lists. */
  readonly withdrawn: boolean;
  readonly poster: string | undefined;
  readonly columns: readonly Column[];
  /** Cells of the Items that count by state, or the withdrawn section's cells. */
  readonly counts: FilesStateCounts;
  /** Items that count and are met, of those that count; null while none counts. */
  readonly tally: Tally | null;
  readonly monitor: SwitchChecked;
  /** Something in it that counts is wanted, or downloading. */
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

/** A cell a search would try to fill, or one a download fills now. */
const opening = (cell: FilesCell) => cell.wanted || cell.state === "downloading";

function monitorOf(items: readonly WorksItem[]): SwitchChecked {
  const on = items.filter(({ monitored }) => monitored).length;
  return on === 0 ? false : on === items.length ? true : "mixed";
}

type Read = {
  readonly cells: ReadonlyMap<string, readonly FilesCell[]>;
  readonly statuses: ReadonlyMap<string, FilesItemStatus>;
  readonly levels: ReadonlyMap<string, FilesLevelRollup>;
};

/** values grouped by their key, in order, leaving out those with none. */
function groupOf<T>(values: readonly T[], keyOf: (value: T) => string | null): Map<string, T[]> {
  const groups = new Map<string, T[]>();
  for (const value of values) {
    const key = keyOf(value);
    const held = key === null ? undefined : groups.get(key);
    if (held !== undefined) {
      held.push(value);
    } else if (key !== null) {
      groups.set(key, [value]);
    }
  }
  return groups;
}

function readOf(cells: FilesCellList): Read {
  return {
    cells: groupOf(cells.cells, (cell) => cell.itemId),
    statuses: new Map(cells.items.map((s) => [s.itemId, s])),
    levels: new Map(cells.levels.map((level) => [level.levelId, level])),
  };
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

/** The withdrawn section's cells: those of each withdrawn level's Items that still hold a file. */
function withdrawnCounts(rollups: readonly (FilesLevelRollup | undefined)[]): FilesStateCounts {
  const counts: Partial<Record<FilesState, number>> = {};
  for (const state of stateOrder) {
    const count = rollups.reduce((sum, rollup) => sum + (rollup?.withdrawn[state] ?? 0), 0);
    if (count > 0) {
      counts[state] = count;
    }
  }
  return counts;
}

function posterOf(type: WorksContentType, level: WorksLevel): string | undefined {
  const art = type.artwork.find(
    (kind) => kind.tier === "level" && kind.depth === level.depth && kind.shape === "poster",
  );
  return level.artwork.find(({ kind }) => kind === art?.key)?.url;
}

const tallyOf = (rollup: FilesLevelRollup | undefined): Tally | null =>
  rollup === undefined || rollup.items === 0
    ? null
    : { met: rollup.itemStatus.met ?? 0, of: rollup.items };

type Section = Pick<LevelView, "id" | "name" | "withdrawn" | "poster">;

const withdrawnId = "withdrawn";

function sectionView(
  context: LevelContext,
  anchor: WorksLevel,
  section: Section,
  items: readonly WorksItem[],
  rollups: readonly (FilesLevelRollup | undefined)[],
  read: Read,
): LevelView | undefined {
  const cellsOf = (item: WorksItem) => read.cells.get(item.id) ?? [];
  const live = items.filter(({ withdrawnAt }) => withdrawnAt === null);
  const gone = items.filter(
    (item) => item.withdrawnAt !== null && cellsOf(item).some((cell) => fileOf(cell) !== null),
  );
  const shown = [...live, ...gone];
  if (shown.length === 0) {
    return undefined;
  }
  const inputs = [
    context,
    ...rollups,
    ...shown.flatMap((item) => [item, read.statuses.get(item.id), ...cellsOf(item)]),
  ];
  const known = levelViews.get(anchor);
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
  const [rollup] = rollups;
  const view: LevelView = {
    ...section,
    columns,
    counts: section.withdrawn ? withdrawnCounts(rollups) : (rollup?.cells ?? {}),
    tally: section.withdrawn ? null : tallyOf(rollup),
    monitor: monitorOf(live),
    opens: live.some((item) => item.monitored && cellsOf(item).some((cell) => opening(cell))),
    entries: foldsOf(context.type, views(live)),
    gone: views(gone),
  };
  levelViews.set(anchor, { inputs, view });
  return view;
}

/**
 * The Work's levels as sections in tree order, asides after the main run,
 * then one section of the Items withdrawn with their levels that still hold
 * files. An unchanged section keeps its view from the previous call.
 */
export function levelsOf(
  context: LevelContext,
  work: Pick<WorksWorkView, "levels" | "items">,
  cells: FilesCellList,
): LevelView[] {
  const read = readOf(cells);
  const byLevel = groupOf(work.items, (item) => item.levelId);
  const holding = work.levels.filter(({ id }) => byLevel.has(id));
  const itemsOf = (levels: readonly WorksLevel[]) =>
    levels.flatMap(({ id }) => byLevel.get(id) ?? []);
  const live = holding
    .filter(({ withdrawnAt }) => withdrawnAt === null)
    .toSorted((a, b) => Number(a.aside) - Number(b.aside))
    .flatMap((level) => {
      const poster = posterOf(context.type, level);
      const section = { id: level.id, name: level.label, withdrawn: false, poster };
      const rollups = [read.levels.get(level.id)];
      return sectionView(context, level, section, itemsOf([level]), rollups, read) ?? [];
    });
  const withdrawn = holding.filter(({ withdrawnAt }) => withdrawnAt !== null);
  const [anchor] = withdrawn;
  if (anchor === undefined) {
    return live;
  }
  const section = {
    id: withdrawnId,
    name: `No longer listed by ${context.type.source}`,
    withdrawn: true,
    poster: undefined,
  };
  const rollups = withdrawn.map(({ id }) => read.levels.get(id));
  const trailing = sectionView(context, anchor, section, itemsOf(withdrawn), rollups, read);
  return trailing === undefined ? live : [...live, trailing];
}
