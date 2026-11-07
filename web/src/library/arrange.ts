import type { WorksContentType } from "../api/gen/model";
import { type FilterGroup, groupsFor, matcherOf } from "./filters";
import type { LibraryWork } from "./rows";
import type { Comparable, Section } from "./sections";
import { type SortDef, sortsFor } from "./sorts";
import type { ArrangeContext, LibraryView } from "./view";

/** What one Library offers, from the Content Types it shows. */
export type Scope = {
  readonly types: readonly WorksContentType[];
  readonly sorts: readonly SortDef[];
  readonly groups: readonly FilterGroup[];
};

const scopes = new WeakMap<readonly WorksContentType[], Scope>();

export function scopeOf(types: readonly WorksContentType[]): Scope {
  const known = scopes.get(types);
  if (known !== undefined) {
    return known;
  }
  const scope = { types, sorts: sortsFor(types), groups: groupsFor(types) };
  scopes.set(types, scope);
  return scope;
}

export type Run = { readonly section: Section; readonly start: number; readonly count: number };

export type Arranged = {
  /** The Works shown, in order. */
  readonly works: readonly LibraryWork[];
  readonly order: readonly string[];
  readonly sections: readonly Run[];
  /** Every Work of the Library's types, before filters and search. */
  readonly total: number;
};

const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: "base" });

function compare(a: Comparable, b: Comparable): number {
  return typeof a === "number" && typeof b === "number"
    ? a - b
    : collator.compare(String(a), String(b));
}

type Keyed = { readonly work: LibraryWork; readonly value: Comparable | undefined };

function sortedBy(
  works: readonly LibraryWork[],
  sort: SortDef,
  reverse: boolean,
  ctx: ArrangeContext,
) {
  const direction = (sort.natural === "asc" ? 1 : -1) * (reverse ? -1 : 1);
  const keyed: Keyed[] = works.map((work) => ({ work, value: sort.value(work, ctx) }));
  return keyed.toSorted((a, b) => {
    if (a.value === undefined || b.value === undefined) {
      if (a.value !== b.value) {
        return a.value === undefined ? 1 : -1;
      }
    } else {
      const order = compare(a.value, b.value) * direction;
      if (order !== 0) {
        return order;
      }
    }
    return collator.compare(a.work.sortLabel, b.work.sortLabel) || (a.work.id < b.work.id ? -1 : 1);
  });
}

function runsOf(keyed: readonly Keyed[], sort: SortDef, ctx: ArrangeContext): Run[] {
  const runs: Run[] = [];
  let current: { section: Section; start: number; count: number } | undefined;
  for (const [index, { value }] of keyed.entries()) {
    const section = sort.section(value, ctx);
    if (current?.section.key === section.key) {
      current.count += 1;
    } else {
      current = { section, start: index, count: 1 };
      runs.push(current);
    }
  }
  return runs;
}

/**
 * Filters the Works of the Library's types, sorts them in the sort's own
 * direction (flipped by Reverse, with Works lacking the value last either
 * way) and cuts them into sections labelled by the value they sort by.
 */
export function arrange(
  works: readonly LibraryWork[],
  scope: Scope,
  view: LibraryView,
  ctx: ArrangeContext,
): Arranged {
  const typeIds = new Set(scope.types.map((type) => type.id));
  const ofTypes = works.filter((work) => typeIds.has(work.type.id));
  const keep = matcherOf(view.filters, scope.groups);
  const { matched } = ctx;
  const shown = ofTypes.filter(
    (work) => (matched === undefined || matched.has(work.id)) && keep(work),
  );
  const sort = scope.sorts.find((each) => each.key === view.sort) ?? scope.sorts[0];
  if (sort === undefined) {
    throw new Error("a Library offers no sort");
  }
  const keyed = sortedBy(shown, sort, view.reverse, ctx);
  const sorted = keyed.map(({ work }) => work);
  return {
    works: sorted,
    order: sorted.map(({ id }) => id),
    sections: runsOf(keyed, sort, ctx),
    total: ofTypes.length,
  };
}
