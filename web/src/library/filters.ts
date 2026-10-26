import type { FilesState, WorksContentType, WorksLibraryRowMonitored } from "../api/gen/model";
import type { LibraryWork } from "./rows";
import { statusMarks } from "./status";
import {
  type ArrangeContext,
  type FilterKey,
  type Filters,
  type ListKind,
  type Pick,
  type SharedField,
  fieldKey,
  monitoredWords,
  noRoot,
  sharedFields,
} from "./view";

/** Keeps the Works holding any of the picked values. */
export type ValuesGroup = {
  readonly kind: "values";
  readonly key: FilterKey;
  readonly label: string;
  readonly valuesOf: (work: LibraryWork) => readonly string[];
  readonly labelOf: (value: string, ctx: ArrangeContext) => string;
  /** The values in the order they are offered; otherwise by their labels. */
  readonly order?: readonly string[];
};

/** Keeps the Works whose number falls inside the picked bounds. */
export type RangeGroup = {
  readonly kind: "range";
  readonly key: FilterKey;
  readonly label: string;
  readonly unit: string;
  readonly numberOf: (work: LibraryWork) => number | undefined;
};

/** Keeps the Works that have it. */
export type FlagGroup = {
  readonly kind: "flag";
  readonly key: FilterKey;
  readonly label: string;
  readonly has: (work: LibraryWork) => boolean;
};

export type FilterGroup = ValuesGroup | RangeGroup | FlagGroup;

const statuses = Object.keys(statusMarks);

const isStatus = (value: string): value is FilesState => Object.hasOwn(statusMarks, value);

const isMonitored = (value: string): value is WorksLibraryRowMonitored =>
  Object.hasOwn(monitoredWords, value);

/** The Works with no library root, as the root filter names them. */
export const rootless = "none";

const coreGroups: readonly FilterGroup[] = [
  {
    kind: "values",
    key: "status",
    label: "Status",
    valuesOf: (work) => (work.status === undefined ? [] : [work.status]),
    labelOf: (value) => (isStatus(value) ? statusMarks[value].word : value),
    order: statuses,
  },
  {
    kind: "values",
    key: "monitored",
    label: "Monitored",
    valuesOf: (work) => [work.monitored],
    labelOf: (value) => (isMonitored(value) ? monitoredWords[value] : value),
    order: Object.keys(monitoredWords),
  },
  {
    kind: "values",
    key: "profile",
    label: "Profile",
    valuesOf: (work) => [work.profileId],
    labelOf: (value, { profiles }) => profiles.get(value) ?? value,
  },
  {
    kind: "values",
    key: "root",
    label: "Library root",
    valuesOf: (work) => [work.rootId ?? rootless],
    labelOf: (value, { roots }) => (value === rootless ? noRoot : (roots.get(value) ?? value)),
  },
  { kind: "flag", key: "needs", label: "Needs you", has: (work) => work.needsYou > 0 },
];

const minuteMs = 60_000;

const fieldGroups: { readonly [K in ListKind]: (field: SharedField) => FilterGroup } = {
  text: ({ key, label }) => ({
    kind: "values",
    key: fieldKey(key),
    label,
    valuesOf: (work) => {
      const value = work.values.get(key);
      return value?.kind === "text" ? [value.text] : [];
    },
    labelOf: (value) => value,
  }),
  texts: ({ key, label }) => ({
    kind: "values",
    key: fieldKey(key),
    label,
    valuesOf: (work) => {
      const value = work.values.get(key);
      return value?.kind === "texts" ? value.texts : [];
    },
    labelOf: (value) => value,
  }),
  number: ({ key, label }) => ({
    kind: "range",
    key: fieldKey(key),
    label,
    unit: "",
    numberOf: (work) => {
      const value = work.values.get(key);
      return value?.kind === "number" ? value.number : undefined;
    },
  }),
  date: ({ key, label }) => ({
    kind: "range",
    key: fieldKey(key),
    label,
    unit: "year",
    numberOf: (work) => {
      const value = work.values.get(key);
      return value?.kind === "date" ? Number(value.date.slice(0, 4)) : undefined;
    },
  }),
  duration: ({ key, label }) => ({
    kind: "range",
    key: fieldKey(key),
    label,
    unit: "minutes",
    numberOf: (work) => {
      const value = work.values.get(key);
      return value?.kind === "duration" ? Math.round(value.ms / minuteMs) : undefined;
    },
  }),
};

/** The filter groups a Library offers: the core ones, then each field its types mark filterable. */
export function groupsFor(types: readonly WorksContentType[]): readonly FilterGroup[] {
  return [
    ...coreGroups,
    ...sharedFields(types, "filter").map((field) => fieldGroups[field.kind](field)),
  ];
}

const inside = (value: number | undefined, { min, max }: { min?: number; max?: number }) =>
  value !== undefined && (min === undefined || value >= min) && (max === undefined || value <= max);

function keeps(work: LibraryWork, group: FilterGroup, pick: Pick): boolean {
  if (group.kind === "values" && pick.kind === "values") {
    const held = group.valuesOf(work);
    return pick.values.some((value) => held.includes(value));
  }
  if (group.kind === "range" && pick.kind === "range") {
    return inside(group.numberOf(work), {
      ...(pick.min !== undefined && { min: pick.min }),
      ...(pick.max !== undefined && { max: pick.max }),
    });
  }
  return group.kind === "flag" && group.has(work);
}

/** A Work passes when every group's pick keeps it: "and" across groups, "or" inside one. */
export function matcherOf(
  filters: Filters,
  groups: readonly FilterGroup[],
): (work: LibraryWork) => boolean {
  const active = groups.flatMap((group) => {
    const pick = filters[group.key];
    return pick === undefined ? [] : [{ group, pick }];
  });
  return (work) => active.every(({ group, pick }) => keeps(work, group, pick));
}

export type Option = { readonly value: string; readonly label: string; readonly count: number };

const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: "base" });

/** The values a group can pick among the Works in view, each with how many hold it. */
export function optionsOf(
  group: ValuesGroup,
  works: readonly LibraryWork[],
  ctx: ArrangeContext,
): readonly Option[] {
  const counts = new Map<string, number>();
  for (const work of works) {
    for (const value of group.valuesOf(work)) {
      counts.set(value, (counts.get(value) ?? 0) + 1);
    }
  }
  const options = [...counts].map(([value, count]) => ({
    value,
    label: group.labelOf(value, ctx),
    count,
  }));
  const { order } = group;
  return order === undefined
    ? options.toSorted((a, b) => collator.compare(a.label, b.label))
    : options.toSorted((a, b) => order.indexOf(a.value) - order.indexOf(b.value));
}
