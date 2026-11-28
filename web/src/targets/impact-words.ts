import type { TargetsImpact, TargetsSubtitles } from "../api/gen/model";
import { bytes, spanWords } from "../format";
import { scansFirst } from "../subtitles/scan-words";
import { languageName } from "../subtitles/settings-words";
import type { Unwanted } from "./use-impact-save";

const grouped = new Intl.NumberFormat("en");

const counted = (count: number, one: string, many: string) =>
  `${String(count)} ${count === 1 ? one : many}`;

const some = (count: number, one: string, many: string) =>
  `${grouped.format(count)} ${count === 1 ? one : many}`;

const languageOf = ({ language, forced }: { language: string; forced: boolean }) =>
  `${forced ? "forced-only " : ""}${languageName(language)}`;

const scanLine = ({ searches, needScan }: { searches: number; needScan: number }) => {
  if (needScan === 0) {
    return [];
  }
  if (needScan === searches) {
    return [`SlipStream scans ${searches === 1 ? "its" : "their"} audio first.`];
  }
  return [scansFirst(needScan)];
};

/** What a change of a Profile's subtitle languages searches for, how long that takes, and what it deletes. */
export function subtitleLines({ gained, dropped, days }: TargetsSubtitles): string[] {
  const searched = gained.filter((want) => want.searches > 0);
  const searches = searched.map((want) =>
    [
      `Saving searches ${languageOf(want)} subtitles for ${some(want.searches, "file", "files")}.`,
      ...scanLine(want),
    ].join(" "),
  );
  const last =
    searched.length > 0 && days > 0
      ? [`At today's downloads, the last of these arrives in about ${some(days, "day", "days")}.`]
      : [];
  const deletes = dropped
    .filter((want) => want.deleted > 0)
    .map(
      (want) =>
        `Saving deletes ${some(want.deleted, `${languageOf(want)} subtitle`, `${languageOf(want)} subtitles`)} SlipStream downloaded. Uploads and subtitles found beside the files stay.`,
    );
  return [...searches, ...last, ...deletes];
}

/** The subtitles a save deletes, which the save confirms. */
export const subtitlesDeleted = (impact: TargetsImpact): number =>
  impact.subtitles.dropped.reduce((count, want) => count + want.deleted, 0);

/** What a change that makes files costs: "SlipStream will make 412 files, about 2 days 13 h and 340 GB." */
export const madeLine = ({ files, seconds, bytes: size }: TargetsImpact["made"]): string =>
  `SlipStream will make ${some(files, "file", "files")}, about ${spanWords(seconds)} and ${bytes(size)}.`;

export function impactLines(impact: TargetsImpact): string[] {
  const { works, ownList, unwanted, renamed, newlyWanted, downloading, made } = impact;
  return [
    works > 0 && `${counted(works, "Work changes", "Works change")}.`,
    ownList > 0 &&
      `${counted(ownList, "Work keeps its own Edition list", "Works keep their own Edition lists")}, so the new list doesn't apply to ${ownList === 1 ? "it" : "them"}.`,
    unwanted.files > 0 &&
      `${counted(unwanted.files, "file would no longer be used by any Target", "files would no longer be used by any Target")} (${bytes(unwanted.bytes)}).`,
    renamed > 0 && `${counted(renamed, "file is renamed", "files are renamed")}.`,
    newlyWanted > 0 &&
      `SlipStream will look for ${counted(newlyWanted, "more file", "more files")}.`,
    made.files > 0 && madeLine(made),
    downloading > 0 &&
      (downloading === 1
        ? "1 download in progress is no longer needed. When it finishes, it waits in Pending imports."
        : `${String(downloading)} downloads in progress are no longer needed. When they finish, they wait in Pending imports.`),
  ]
    .filter((line) => line !== false)
    .concat(subtitleLines(impact.subtitles));
}

export function moreRenamed(impact: TargetsImpact): string | undefined {
  const listed = impact.renames.reduce((count, work) => count + work.files.length, 0);
  const more = impact.renamed - listed;
  if (more === 0) {
    return undefined;
  }
  return more === 1 ? "1 more file is renamed." : `${grouped.format(more)} more files are renamed.`;
}

export function saveLabel(action: string, impact: TargetsImpact, unwanted: Unwanted): string {
  const count = impact.unwanted.files;
  const subtitles = subtitlesDeleted(impact);
  const files = count === 0 ? undefined : `${unwanted} ${counted(count, "file", "files")}`;
  const deletes =
    subtitles === 0 ? undefined : `delete ${counted(subtitles, "subtitle", "subtitles")}`;
  if (files === undefined) {
    return deletes === undefined ? action : `${action} and ${deletes}`;
  }
  return deletes === undefined ? `${action} and ${files}` : `${action}, ${files} and ${deletes}`;
}
