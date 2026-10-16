import type { Level, LogRow } from "./buffer";

export type LogFilter = {
  readonly min: Level;
  readonly subsystem: string | null;
  readonly search: string;
};

export const noFilter: LogFilter = { min: "debug", subsystem: null, search: "" };

const rank: { readonly [L in Level]: number } = { debug: 0, info: 1, warn: 2, error: 3 };

export const isFiltered = (filter: LogFilter): boolean =>
  filter.min !== "debug" || filter.subsystem !== null || filter.search.trim() !== "";

/** The rows a filter keeps. Dividers always stay, so a gap in the log stays visible. */
export function visibleRows(rows: readonly LogRow[], filter: LogFilter): readonly LogRow[] {
  if (!isFiltered(filter)) {
    return rows;
  }
  const min = rank[filter.min];
  const needle = filter.search.trim().toLowerCase();
  return rows.filter(
    (row) =>
      row.kind === "divider" ||
      (rank[row.level] >= min &&
        (filter.subsystem === null || row.subsystem === filter.subsystem) &&
        row.haystack.includes(needle)),
  );
}

export function subsystemsOf(rows: readonly LogRow[]): readonly string[] {
  const names = new Set<string>();
  for (const row of rows) {
    if (row.kind === "line") {
      names.add(row.subsystem);
    }
  }
  return [...names].toSorted((a, b) => a.localeCompare(b));
}
