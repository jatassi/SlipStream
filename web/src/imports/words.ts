import type {
  ImportsLead,
  ImportsReason,
  ImportsReasonCode,
  WorksContentType,
} from "../api/gen/model";
import type { RootEntry } from "./root-entry";
import type { Noun } from "../format";

const counted = new Intl.NumberFormat("en");

const lower = ({ one, many }: Noun): Noun => ({ one: one.toLowerCase(), many: many.toLowerCase() });

const countOf = (count: number, { one, many }: Noun) =>
  `${counted.format(count)} ${count === 1 ? one : many}`;

/** A title with its year, such as Heat (1995). */
export const named = (title: string, year: number | null | undefined): string =>
  year === null || year === undefined ? title : `${title} (${year})`;

function given<T>(value: T | undefined, what: string): T {
  if (value === undefined) {
    throw new Error(`the reason carries no ${what}`);
  }
  return value;
}

/** How one reason reads for `files` of an entry's files that share it. */
type Wording = (reason: ImportsReason, type: WorksContentType, files: number) => string;

const reasonWords: { readonly [C in ImportsReasonCode]: Wording } = {
  several: ({ title, year, count }, type) =>
    `${countOf(given(count, "count"), lower(type.work))} match ${named(given(title, "title"), year)}.${year === undefined ? " The name has no year." : ""}`,
  no_match: ({ title, year }, type) =>
    `No ${lower(type.work).one} matches ${named(given(title, "title"), year)}.`,
  no_year: ({ title }, type) =>
    `One ${lower(type.work).one} matches ${given(title, "title")}, but the name has no year.`,
  elsewhere: ({ title, year, root }) =>
    `${named(given(title, "title"), year)} is in the ${given(root, "root").label}. A Work's files all live in one root. Move these files there, or ignore them.`,
  unlisted: ({ title, year }, type) =>
    `${type.source} no longer lists ${named(given(title, "title"), year)}.`,
  no_number: (_reason, type, files) =>
    files === 1
      ? `The file name has no ${lower(type.item).one} number.`
      : `${counted.format(files)} file names have no ${lower(type.item).one} number.`,
  no_such_item: ({ names }, type) =>
    `This ${lower(type.work).one} has no ${lower(type.item).one} ${given(names, "names")}.`,
  several_items: (_reason, type, files) =>
    files === 1
      ? `The file holds several ${lower(type.item).many}.`
      : `${counted.format(files)} files each hold several ${lower(type.item).many}.`,
  unreadable: (_reason, _type, files) =>
    files === 1
      ? "ffprobe can't read this file. Replace it, then scan again."
      : `ffprobe can't read ${counted.format(files)} of these files. Replace them, then scan again.`,
  left: (_reason, type) =>
    `This ${lower(type.work).one} was deleted from the library and its files were left on disk.`,
  no_rule: (_reason, type, files) =>
    files === 1
      ? `SlipStream can't tell which ${lower(type.item).one} this file holds.`
      : `SlipStream can't tell which ${lower(type.item).one} each of ${counted.format(files)} files holds.`,
};

/** Why one file waits, in plain words. */
export const reasonOf = (reason: ImportsReason, type: WorksContentType): string =>
  reasonWords[reason.code](reason, type, 1);

/** Why an entry waits: one sentence per reason its files share, in file order. */
export function reasonsOf(entry: RootEntry, type: WorksContentType): string[] {
  const shared = new Map<string, { reason: ImportsReason; files: number }>();
  for (const { reason } of entry.files) {
    const key = JSON.stringify(reason);
    const seen = shared.get(key);
    shared.set(key, { reason, files: (seen?.files ?? 0) + 1 });
  }
  return [
    ...new Set(
      [...shared.values()].map(({ reason, files }) =>
        reasonWords[reason.code](reason, type, files),
      ),
    ),
  ];
}

/** An entry waits on a Work in another root, so it offers only Ignore. */
export const isElsewhere = (entry: RootEntry): boolean =>
  entry.files.some(({ reason }) => reason.code === "elsewhere");

/** Its suggestion can be added, or take its files, from here. */
export const isAddable = (entry: RootEntry): boolean =>
  entry.suggestion !== null && !isElsewhere(entry);

/** "14 folders need you", the Library's link to Pending imports. */
export const needYouWords = (count: number): string =>
  `${countOf(count, { one: "folder", many: "folders" })} ${count === 1 ? "needs" : "need"} you`;

/** "The last scan of the Films library root on home added 398 films. 14 folders need you." */
export function leadWords(lead: ImportsLead, type: WorksContentType): string {
  const works = lower(type.work);
  const added =
    lead.added === 0 ? `didn't add any ${works.many}` : `added ${countOf(lead.added, works)}`;
  const needs = lead.needsYou === 0 ? "No folders need you." : `${needYouWords(lead.needsYou)}.`;
  return `The last scan of the ${lead.root.label} ${added}. ${needs}`;
}

export const filesWords = (count: number): string => countOf(count, { one: "file", many: "files" });
