import type { FilesState, WorksContentType, WorksLibraryRowMonitored } from "../api/gen/model";
import type { LibraryWork } from "./rows";
import {
  type Comparable,
  type Section,
  bytesSection,
  dateSections,
  durationSections,
  letterSection,
  namedSections,
  ratingSection,
} from "./sections";
import { statusMarks } from "./status";
import {
  type ArrangeContext,
  type CoreSort,
  type ListKind,
  type SharedField,
  type SortKey,
  fieldKey,
  monitoredWords,
  noRoot,
  rated,
  sharedFields,
} from "./view";

export type SortDef = {
  readonly key: SortKey;
  readonly label: string;
  readonly natural: "asc" | "desc";
  /** Undefined sorts last in either direction. */
  readonly value: (work: LibraryWork, ctx: ArrangeContext) => Comparable | undefined;
  /** Sections are labelled and ordered by the same value the sort uses. */
  readonly section: (value: Comparable | undefined, ctx: ArrangeContext) => Section;
};

/** Failures first, then what SlipStream can still act on, as Needs you ranks them. */
const statusOrder: readonly FilesState[] = [
  "failed",
  "upgradable",
  "missing",
  "downloading",
  "met",
  "unreleased",
];

const monitoredOrder: readonly WorksLibraryRowMonitored[] = ["all", "some", "none"];

/** Sorts by a word's place in `order` and sections by the word. */
function ranked<W extends string>(order: readonly W[], words: (word: W) => string, none: string) {
  const sections = namedSections(none);
  return {
    rank: (word: W | undefined) => (word === undefined ? undefined : order.indexOf(word)),
    section: (value: Comparable | undefined) => {
      const word = typeof value === "number" ? order[value] : undefined;
      return sections(word === undefined ? undefined : words(word));
    },
  };
}

const byStatus = ranked(statusOrder, (status) => statusMarks[status].word, "No status");
const byMonitored = ranked(monitoredOrder, (monitored) => monitoredWords[monitored], "");

/** The local day and time, which sorts as text and opens with the day dateSections reads. */
function localStamp(date: Date): string {
  const two = (part: number) => String(part).padStart(2, "0");
  return `${date.getFullYear()}-${two(date.getMonth() + 1)}-${two(date.getDate())}T${two(date.getHours())}:${two(date.getMinutes())}:${two(date.getSeconds())}`;
}

const byDate = (value: Comparable | undefined, { today }: ArrangeContext) =>
  dateSections(today)(value);

const coreSorts: { readonly [K in CoreSort]: SortDef } = {
  title: {
    key: "title",
    label: "Title",
    natural: "asc",
    value: (work) => work.sortLabel,
    section: letterSection,
  },
  added: {
    key: "added",
    label: "Date added",
    natural: "desc",
    value: (work) => localStamp(work.added),
    section: byDate,
  },
  available: {
    key: "available",
    label: "Availability date",
    natural: "desc",
    value: (work) => work.availableOn,
    section: byDate,
  },
  size: {
    key: "size",
    label: "Size on disk",
    natural: "desc",
    value: (work) => (work.files === 0 ? undefined : work.bytes),
    section: bytesSection,
  },
  status: {
    key: "status",
    label: "Status",
    natural: "asc",
    value: (work) => byStatus.rank(work.status),
    section: byStatus.section,
  },
  profile: {
    key: "profile",
    label: "Profile",
    natural: "asc",
    value: (work, { profiles }) => profiles.get(work.profileId),
    section: namedSections("No Profile"),
  },
  root: {
    key: "root",
    label: "Library root",
    natural: "asc",
    value: (work, { roots }) => (work.rootId === undefined ? undefined : roots.get(work.rootId)),
    section: namedSections(noRoot),
  },
  monitored: {
    key: "monitored",
    label: "Monitored",
    natural: "asc",
    value: (work) => byMonitored.rank(work.monitored),
    section: byMonitored.section,
  },
  rating: {
    key: "rating",
    label: "Rating",
    natural: "desc",
    value: ({ rating }) => (rating === undefined ? undefined : (rating.value / rating.scale) * 10),
    section: ratingSection,
  },
};

const noneOf = (label: string) => `No ${label.toLowerCase()}`;

const fieldSorts: {
  readonly [K in ListKind]: (field: SharedField) => Pick<SortDef, "natural" | "value" | "section">;
} = {
  text: ({ key, label }) => ({
    natural: "asc",
    value: (work) => {
      const value = work.values.get(key);
      return value?.kind === "text" ? value.text : undefined;
    },
    section: namedSections(noneOf(label)),
  }),
  texts: ({ key, label }) => ({
    natural: "asc",
    value: (work) => {
      const value = work.values.get(key);
      return value?.kind === "texts" ? value.texts[0] : undefined;
    },
    section: namedSections(noneOf(label)),
  }),
  number: ({ key, label }) => ({
    natural: "desc",
    value: (work) => {
      const value = work.values.get(key);
      return value?.kind === "number" ? value.number : undefined;
    },
    section: namedSections(noneOf(label)),
  }),
  date: ({ key }) => ({
    natural: "desc",
    value: (work) => {
      const value = work.values.get(key);
      return value?.kind === "date" ? value.date : undefined;
    },
    section: byDate,
  }),
  duration: ({ key, label }) => ({
    natural: "desc",
    value: (work) => {
      const value = work.values.get(key);
      return value?.kind === "duration" ? value.ms : undefined;
    },
    section: durationSections(noneOf(label)),
  }),
};

/** The sorts a Library offers: the core ones, then each field its types mark sortable. */
export function sortsFor(types: readonly WorksContentType[]): readonly SortDef[] {
  const core = Object.values(coreSorts).filter((sort) => sort.key !== "rating" || rated(types));
  const fields = sharedFields(types, "sort").map((field) => ({
    key: fieldKey(field.key),
    label: field.label,
    ...fieldSorts[field.kind](field),
  }));
  return [...core, ...fields];
}
