import type { WorksContentType } from "../api/gen/model";
import { scopeOf } from "./arrange";
import type { FilterGroup } from "./filters";
import {
  type ColumnKey,
  type FilterKey,
  type Filters,
  type LibraryView,
  type Pick,
  type SortKey,
  columnKeys,
  defaultView,
  isFieldKey,
  keyOfField,
  layouts,
  posterSizes,
} from "./view";

/**
 * A Library's view in its address, such as `view=table`, `cols=status,field.year`,
 * `sort=field.year`, `rev=1`, `status=missing,failed`, `needs=1`,
 * `f.genres=Crime~Drama` and `f.year=1990..1999`. Anything left out is the
 * default.
 */
export type LibrarySearch = Readonly<Record<string, string | 1>>;

const texts = ["view", "size", "cols", "sort", "status", "monitored", "profile", "root", "q"];
const flags = ["rev", "needs"];

const textOf = (value: unknown) =>
  typeof value === "string" || typeof value === "number" ? String(value) : undefined;

const flagOf = (value: unknown) => value === 1 || value === "1" || value === true;

/** Reads the address's view keys, keeping each as text, and drops every other key. */
export function librarySearch(raw: unknown): LibrarySearch {
  if (typeof raw !== "object" || raw === null) {
    return {};
  }
  return Object.fromEntries(
    Object.entries(raw).flatMap(([key, value]): [string, string | 1][] => {
      if (flags.includes(key)) {
        return flagOf(value) ? [[key, 1]] : [];
      }
      const text = texts.includes(key) || key.startsWith("f.") ? textOf(value) : undefined;
      return text === undefined || text === "" ? [] : [[key, text]];
    }),
  );
}

/** The address keys that set the view, which a remembered view never carries the search of. */
export const viewKeys = (search: LibrarySearch): string[] =>
  Object.keys(search).filter((key) => key !== "q");

const paramOf = (key: FilterKey) => (isFieldKey(key) ? `f.${keyOfField(key)}` : key);

// Field values are free text, so they part on "~"; ids and states never hold a comma.
const separatorOf = (key: FilterKey) => (isFieldKey(key) ? "~" : ",");

const rangePattern = /^(-?\d+(?:\.\d+)?)?\.\.(-?\d+(?:\.\d+)?)?$/u;

function pickOf(group: FilterGroup, text: string | undefined): Pick | undefined {
  if (text === undefined) {
    return undefined;
  }
  if (group.kind === "flag") {
    return { kind: "flag" };
  }
  if (group.kind === "values") {
    const values = [...new Set(text.split(separatorOf(group.key)).filter((value) => value !== ""))];
    return values.length === 0 ? undefined : { kind: "values", values };
  }
  const exact = Number(text);
  if (Number.isFinite(exact)) {
    return { kind: "range", min: exact, max: exact };
  }
  const [, min, max] = rangePattern.exec(text) ?? [];
  return min === undefined && max === undefined
    ? undefined
    : { kind: "range", min: min === undefined ? undefined : Number(min), max: max === undefined ? undefined : Number(max) };
}

function textOfPick(key: FilterKey, pick: Pick): string | 1 {
  if (pick.kind === "flag") {
    return 1;
  }
  if (pick.kind === "values") {
    return pick.values.join(separatorOf(key));
  }
  return `${pick.min ?? ""}..${pick.max ?? ""}`;
}

const textAt = (search: LibrarySearch, key: string): string | undefined => {
  const value = search[key];
  return typeof value === "string" ? value : undefined;
};

function filtersOf(search: LibrarySearch, groups: readonly FilterGroup[]): Filters {
  return Object.fromEntries(
    groups.flatMap((group) => {
      const value = search[paramOf(group.key)];
      const pick = pickOf(group, value === undefined ? undefined : String(value));
      return pick === undefined ? [] : [[group.key, pick]];
    }),
  );
}

const oneOf = <T extends string>(choices: readonly T[], value: string | undefined, fallback: T): T =>
  choices.find((choice) => choice === value) ?? fallback;

/** The view an address names for these types, dropping what they no longer declare. */
export function decodeView(search: LibrarySearch, types: readonly WorksContentType[]): LibraryView {
  const fallback = defaultView(types);
  const { sorts, groups } = scopeOf(types);
  const known = columnKeys(types);
  const picked = textAt(search, "cols")?.split(",");
  const columns = picked === undefined ? undefined : known.filter((key) => picked.includes(key));
  return {
    layout: oneOf(layouts, textAt(search, "view"), fallback.layout),
    size: oneOf(posterSizes, textAt(search, "size"), fallback.size),
    columns: columns ?? fallback.columns,
    sort: oneOf<SortKey>(sorts.map(({ key }) => key), textAt(search, "sort"), fallback.sort),
    reverse: search["rev"] === 1,
    filters: filtersOf(search, groups),
    q: textAt(search, "q") ?? "",
  };
}

const sameColumns = (a: readonly ColumnKey[], b: readonly ColumnKey[]) =>
  a.length === b.length && a.every((key, index) => key === b[index]);

/** The address of a view: only what differs from the default. */
export function encodeView(view: LibraryView, types: readonly WorksContentType[]): LibrarySearch {
  const fallback = defaultView(types);
  const { groups } = scopeOf(types);
  const filters = groups.flatMap((group) => {
    const pick = view.filters[group.key];
    return pick === undefined ? [] : [[paramOf(group.key), textOfPick(group.key, pick)]];
  });
  return {
    ...(view.layout !== fallback.layout && { view: view.layout }),
    ...(view.size !== fallback.size && { size: view.size }),
    ...(!sameColumns(view.columns, fallback.columns) && { cols: view.columns.join(",") }),
    ...(view.sort !== fallback.sort && { sort: view.sort }),
    ...(view.reverse && { rev: 1 }),
    ...Object.fromEntries(filters),
    ...(view.q !== "" && { q: view.q }),
  };
}
