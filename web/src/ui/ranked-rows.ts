/**
 * A Ranked list's rows in order. An item tied with the one above shares its
 * rank; a marker, such as the cutoff, sits between ranks and holds none.
 */
export type RankedRow<T> =
  | { readonly kind: "item"; readonly id: string; readonly value: T; readonly tiedAbove: boolean }
  | { readonly kind: "marker"; readonly id: string; readonly label: string };

export type Rows<T> = readonly RankedRow<T>[];

export type Step = -1 | 1;

// A unit is what one step passes over: a tier of tied items, or a marker.
type Unit<T> = RankedRow<T>[];

function unitsOf<T>(rows: Rows<T>): Unit<T>[] {
  const units: Unit<T>[] = [];
  for (const row of rows) {
    const last = units.at(-1);
    if (row.kind === "item" && row.tiedAbove && last?.[0]?.kind === "item") {
      last.push(row);
    } else {
      units.push([row]);
    }
  }
  return units;
}

const rowsOf = <T>(units: readonly Unit<T>[]): RankedRow<T>[] =>
  units.flatMap((unit) =>
    unit.map((row, at) => (row.kind === "item" ? { ...row, tiedAbove: at > 0 } : row)),
  );

/** The rows with each tier's first item untied, as every edit leaves them. */
export const settle = <T>(rows: Rows<T>): RankedRow<T>[] => rowsOf(unitsOf(rows));

const holding = <T>(units: readonly Unit<T>[], id: string) =>
  units.findIndex((unit) => unit.some((row) => row.id === id));

/**
 * Whether a step can move the row: a tied item can always step out of its
 * tier, and anything else needs a unit to pass in that direction.
 */
export function canStep<T>(rows: Rows<T>, id: string, step: Step): boolean {
  const units = unitsOf(rows);
  const at = holding(units, id);
  return (units[at]?.length ?? 0) > 1 || (at + step >= 0 && at + step < units.length);
}

/**
 * One step up or down, as the Move buttons and the handle's arrow keys take
 * it. A tied item steps out of its tier to just above or below it; an untied
 * item or a marker passes the whole tier or marker next to it, so a step
 * never splits a tier.
 */
export function stepRow<T>(rows: Rows<T>, id: string, step: Step): RankedRow<T>[] {
  const units = unitsOf(rows);
  const at = holding(units, id);
  const unit = units[at];
  const row = unit?.find((candidate) => candidate.id === id);
  if (unit === undefined || row === undefined) {
    throw new Error(`no ranked row ${id}`);
  }
  if (unit.length > 1) {
    const rest = unit.filter((candidate) => candidate !== row);
    units.splice(at, 1, ...(step === -1 ? [[row], rest] : [rest, [row]]));
    return rowsOf(units);
  }
  const passed = units[at + step];
  if (passed === undefined) {
    return [...rows];
  }
  units[at] = passed;
  units[at + step] = unit;
  return rowsOf(units);
}

/** Ties an item with the row above, or unties it. A tie needs an item above. */
export function tieRow<T>(rows: Rows<T>, id: string, tied: boolean): RankedRow<T>[] {
  return settle(
    rows.map((row) => (row.kind === "item" && row.id === id ? { ...row, tiedAbove: tied } : row)),
  );
}

export const canTie = <T>(rows: Rows<T>, id: string): boolean => {
  const at = rows.findIndex((row) => row.id === id);
  return rows[at]?.kind === "item" && rows[at - 1]?.kind === "item";
};

/**
 * Where a drag lets go. `order` is the drag's order of ids, the dropped row
 * moved and the rest as they were. An item let go inside a tier joins it. A
 * marker let go inside a tier carries on past it, so it never splits one.
 */
export function dropRow<T>(rows: Rows<T>, order: readonly string[], id: string): RankedRow<T>[] {
  const from = rows.findIndex((row) => row.id === id);
  const row = rows[from];
  const to = order.indexOf(id);
  if (row === undefined || to === -1) {
    throw new Error(`no ranked row ${id}`);
  }
  if (to === from) {
    return [...rows];
  }
  const rest = settle(rows.filter((candidate) => candidate !== row));
  const inTier = (at: number) => {
    const next = rest[at];
    return at > 0 && next?.kind === "item" && next.tiedAbove;
  };
  let at = to;
  if (row.kind === "marker") {
    while (inTier(at)) {
      at += to > from ? 1 : -1;
    }
  }
  const placed = row.kind === "item" ? { ...row, tiedAbove: inTier(at) } : row;
  return settle([...rest.slice(0, at), placed, ...rest.slice(at)]);
}

/** Each item's rank, counting tiers, so tied items share one. */
export function ranksOf<T>(rows: Rows<T>): ReadonlyMap<string, number> {
  const ranks = new Map<string, number>();
  let rank = 0;
  for (const unit of unitsOf(rows)) {
    if (unit[0]?.kind !== "item") {
      continue;
    }
    rank += 1;
    for (const row of unit) {
      ranks.set(row.id, rank);
    }
  }
  return ranks;
}

/** The items' values by tier, highest first; markers are left out. */
export const tiersOf = <T>(rows: Rows<T>): T[][] =>
  unitsOf(rows).flatMap((unit) => {
    const values = unit.flatMap((row) => (row.kind === "item" ? [row.value] : []));
    return values.length > 0 ? [values] : [];
  });

/** What a move announces: an item's new rank, or where a marker now sits. */
export function moveSentence<T>(
  rows: Rows<T>,
  id: string,
  nameOf: (row: RankedRow<T>) => string,
): string {
  const at = rows.findIndex((row) => row.id === id);
  const row = rows[at];
  if (row === undefined) {
    throw new Error(`no ranked row ${id}`);
  }
  if (row.kind === "item") {
    const ranks = ranksOf(rows);
    return `${nameOf(row)} moved to rank ${String(ranks.get(id))} of ${String(Math.max(...ranks.values()))}.`;
  }
  const above = rows[at - 1];
  const below = rows[at + 1];
  if (above !== undefined) {
    return `${row.label} moved below ${nameOf(above)}.`;
  }
  return `${row.label} moved above ${below === undefined ? "every row" : nameOf(below)}.`;
}
