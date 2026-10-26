import type {
  Bare,
  FilesState,
  FilesStateCounts,
  WorksContentType,
  WorksFieldSpecKind,
  WorksLibraryRow,
  WorksLibraryRowMonitored,
  WorksRowRating,
} from "../api/gen/model";

/** A year, a year and month, or a full date, as "2021", "2021-04" or "2021-04-09"; it sorts as text. */
export type PartialDate = string & { readonly __brand: "PartialDate" };

/** A listed field's value, typed by its descriptor's kind. */
export type ListValue =
  | { readonly kind: "text"; readonly text: string }
  | { readonly kind: "number"; readonly number: number }
  | { readonly kind: "date"; readonly date: PartialDate }
  | { readonly kind: "duration"; readonly ms: number }
  | { readonly kind: "texts"; readonly texts: readonly string[] };

/** One Library row, parsed against its Content Type. */
export type LibraryWork = {
  readonly id: string;
  readonly type: WorksContentType;
  readonly key: string;
  readonly label: string;
  readonly sortLabel: string;
  readonly added: Date;
  /** Its metadata source no longer lists it. */
  readonly gone: boolean;
  readonly monitored: WorksLibraryRowMonitored;
  readonly profileId: string;
  readonly rootId: string | undefined;
  /** The artwork route for its poster, without a size. */
  readonly poster: string | undefined;
  readonly rating: WorksRowRating | undefined;
  readonly availableOn: PartialDate | undefined;
  readonly bytes: number;
  readonly files: number;
  readonly kept: number;
  /** Undefined for a Work with no cell. */
  readonly status: FilesState | undefined;
  readonly cells: FilesStateCounts;
  /** Its wanted cells that failed, for the Failed chip. */
  readonly failed: number;
  readonly items: number;
  readonly itemStatus: FilesStateCounts;
  readonly needsYou: number;
  readonly values: ReadonlyMap<string, ListValue>;
};

const partialDate = /^\d{4}(?:-\d{2}(?:-\d{2})?)?$/u;

const asPartialDate = (text: string): PartialDate | undefined =>
  // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the brand is given only after the pattern matched
  partialDate.test(text) ? (text as PartialDate) : undefined;

type ListKind = ListValue["kind"];

const parsers: { readonly [K in ListKind]: (value: Bare) => ListValue | undefined } = {
  text: (value) => (typeof value === "string" ? { kind: "text", text: value } : undefined),
  number: (value) => (typeof value === "number" ? { kind: "number", number: value } : undefined),
  date: (value) => {
    const date = typeof value === "string" ? asPartialDate(value) : undefined;
    return date === undefined ? undefined : { kind: "date", date };
  },
  duration: (value) => (typeof value === "number" ? { kind: "duration", ms: value } : undefined),
  texts: (value) => (Array.isArray(value) ? { kind: "texts", texts: value } : undefined),
};

const isListKind = (kind: WorksFieldSpecKind): kind is ListKind => Object.hasOwn(parsers, kind);

const listedKinds = new WeakMap<WorksContentType, ReadonlyMap<string, ListKind>>();

/** The kind of each Work field the type puts in the Library, by key. */
function listedOf(type: WorksContentType): ReadonlyMap<string, ListKind> {
  const known = listedKinds.get(type);
  if (known !== undefined) {
    return known;
  }
  const listed = new Map<string, ListKind>();
  for (const { key, kind, tier, list } of type.fields) {
    if (tier === "work" && (list.column || list.sort || list.filter) && isListKind(kind)) {
      listed.set(key, kind);
    }
  }
  listedKinds.set(type, listed);
  return listed;
}

function valuesOf(row: WorksLibraryRow, type: WorksContentType): ReadonlyMap<string, ListValue> {
  const listed = listedOf(type);
  const values = new Map<string, ListValue>();
  for (const [key, value] of Object.entries(row.values)) {
    const kind = listed.get(key);
    const parsed = kind === undefined ? undefined : parsers[kind](value);
    if (parsed !== undefined) {
      values.set(key, parsed);
    }
  }
  return values;
}

const parsed = new WeakMap<WorksContentType, WeakMap<WorksLibraryRow, LibraryWork>>();

/**
 * Parses a wire row against its Content Type: each value typed by its
 * field's kind, and a key the type no longer lists dropped. The answer is
 * cached per wire row and type, so a splice parses only the rows it
 * replaced.
 */
export function parseRow(row: WorksLibraryRow, type: WorksContentType): LibraryWork {
  const cache = parsed.get(type) ?? new WeakMap<WorksLibraryRow, LibraryWork>();
  parsed.set(type, cache);
  const known = cache.get(row);
  if (known !== undefined) {
    return known;
  }
  const work: LibraryWork = {
    id: row.id,
    type,
    key: row.key,
    label: row.label,
    sortLabel: row.sortLabel,
    added: new Date(row.addedAt),
    gone: row.goneUpstreamAt !== null,
    monitored: row.monitored,
    profileId: row.profileId,
    rootId: row.rootId ?? undefined,
    poster: row.posterUrl ?? undefined,
    rating: row.rating ?? undefined,
    availableOn: row.availableOn === null ? undefined : asPartialDate(row.availableOn),
    bytes: row.bytes,
    files: row.files,
    kept: row.kept,
    status: row.status ?? undefined,
    cells: row.cells,
    failed: row.failed,
    items: row.items,
    itemStatus: row.itemStatus,
    needsYou: row.needsYou,
    values: valuesOf(row, type),
  };
  cache.set(row, work);
  return work;
}
