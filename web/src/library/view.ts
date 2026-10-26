import type {
  WorksContentType,
  WorksLibraryRowMonitored,
  WorksListing,
} from "../api/gen/model";
import type { ListValue } from "./rows";

export const layouts = ["grid", "table"] as const;
export type Layout = (typeof layouts)[number];

export const posterSizes = ["small", "medium", "large"] as const;
export type PosterSize = (typeof posterSizes)[number];

/** A Content Type's listed field, by its key. */
export type FieldKey = `field.${string}`;

export type CoreSort =
  | "title"
  | "added"
  | "available"
  | "size"
  | "status"
  | "profile"
  | "root"
  | "monitored"
  | "rating";
export type SortKey = CoreSort | FieldKey;

export type CoreColumn =
  | "status"
  | "profile"
  | "root"
  | "size"
  | "files"
  | "added"
  | "available"
  | "monitored"
  | "rating";
export type ColumnKey = CoreColumn | FieldKey;

export type CoreFilter = "status" | "monitored" | "profile" | "root" | "needs";
export type FilterKey = CoreFilter | FieldKey;

/** What one filter group keeps: any of some values, a range, or the Works that have a flag. */
export type Pick =
  | { readonly kind: "values"; readonly values: readonly string[] }
  | { readonly kind: "range"; readonly min: number | undefined; readonly max: number | undefined }
  | { readonly kind: "flag" };

export type Filters = { readonly [K in FilterKey]?: Pick };

export type LibraryView = {
  readonly layout: Layout;
  readonly size: PosterSize;
  readonly columns: readonly ColumnKey[];
  readonly sort: SortKey;
  /** Flips the sort's own direction, which each sort sets. */
  readonly reverse: boolean;
  readonly filters: Filters;
  readonly q: string;
};

export type ListKind = ListValue["kind"];

/** A field every Content Type in view lists, with the same key and kind. */
export type SharedField = { readonly key: string; readonly label: string; readonly kind: ListKind };

const listKinds: ReadonlySet<string> = new Set(["text", "number", "date", "duration", "texts"]);

const isListKind = (kind: string): kind is ListKind => listKinds.has(kind);

/**
 * The fields the Library can use for `use` across every type in view: on
 * one type's Library its own, on "All" only those each type declares alike.
 */
export function sharedFields(
  types: readonly WorksContentType[],
  use: keyof WorksListing,
): readonly SharedField[] {
  const [first, ...rest] = types;
  if (first === undefined) {
    return [];
  }
  const listed = (type: WorksContentType) =>
    type.fields.filter((field) => field.tier === "work" && field.list[use]);
  return listed(first).flatMap(({ key, label, kind }) =>
    isListKind(kind) &&
    rest.every((type) => listed(type).some((field) => field.key === key && field.kind === kind))
      ? [{ key, label, kind }]
      : [],
  );
}

export const fieldKey = (key: string): FieldKey => `field.${key}`;

export const keyOfField = (key: FieldKey): string => key.slice("field.".length);

export const isFieldKey = (key: string): key is FieldKey => key.startsWith("field.");

/** What arranging needs beyond the rows: today, names by id, and the search box's matches. */
export type ArrangeContext = {
  readonly today: Date;
  readonly profiles: ReadonlyMap<string, string>;
  readonly roots: ReadonlyMap<string, string>;
  /** The Works the search box matched; undefined while it is empty. */
  readonly matched: ReadonlySet<string> | undefined;
};

export const monitoredWords: { readonly [M in WorksLibraryRowMonitored]: string } = {
  all: "Monitored",
  some: "Partly monitored",
  none: "Not monitored",
};

export const noRoot = "No library root";

const coreColumns: readonly CoreColumn[] = [
  "status",
  "monitored",
  "profile",
  "root",
  "size",
  "files",
  "added",
  "available",
  "rating",
];

/** Whether every type in view shows ratings, so the Library can sort and show them. */
export const rated = (types: readonly WorksContentType[]): boolean =>
  types.length > 0 && types.every((type) => type.ratings.length > 0);

/** The table's columns in the order they stand: the core ones, then each listed field. */
export function columnKeys(types: readonly WorksContentType[]): readonly ColumnKey[] {
  return [
    ...coreColumns.filter((key) => key !== "rating" || rated(types)),
    ...sharedFields(types, "column").map(({ key }) => fieldKey(key)),
  ];
}

const shownAtFirst: ReadonlySet<ColumnKey> = new Set(["status", "profile", "size", "added"]);

/** A Library's view before anyone changes it: a grid of medium posters by title. */
export function defaultView(types: readonly WorksContentType[]): LibraryView {
  return {
    layout: "grid",
    size: "medium",
    columns: columnKeys(types).filter((key) => shownAtFirst.has(key) || isFieldKey(key)),
    sort: "title",
    reverse: false,
    filters: {},
    q: "",
  };
}
