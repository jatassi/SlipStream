import type { FilesRowBlock } from "../api/gen/model";

const grouped = new Intl.NumberFormat("en");
const listed = new Intl.ListFormat("en", { type: "conjunction" });

export const counted = (count: number, one: string, many: string) =>
  `${grouped.format(count)} ${count === 1 ? one : many}`;

export const filesWords = (count: number) => counted(count, "file", "files");

/** What confirming does, as ticked: in one Work, the Work goes unsaid. */
export function leadWords({
  files,
  works,
  saving,
  oneWork,
}: {
  files: number;
  works: number;
  saving: boolean;
  oneWork: boolean;
}): string {
  if (files === 0) {
    return saving ? "Saving renames no files." : "No files are ticked to rename.";
  }
  const where = oneWork ? "" : ` in ${counted(works, "Work", "Works")}`;
  return `${saving ? "Saving renames" : "This renames"} ${filesWords(files)}${where}.`;
}

export const nothingWords = "Every file matches the templates.";

export const blockedWords = (count: number) =>
  count === 1
    ? "1 file can't be renamed now. Open its Work to see why."
    : `${grouped.format(count)} files can't be renamed now. Open their Works to see why.`;

export const movingWords = (count: number) =>
  count === 1
    ? "1 file is still moving and isn't listed."
    : `${grouped.format(count)} files are still moving and aren't listed.`;

export const changedWords = (count: number) =>
  count === 1
    ? "1 Work changed since you reviewed it. Check it and rename it again."
    : `${grouped.format(count)} Works changed since you reviewed them. Check them and rename them again.`;

export const renamingWords = (count: number) => `Renaming ${filesWords(count)}…`;

/** A Work's line: what it renames as ticked, and what can't be renamed. */
export function workWords(ticked: number, rename: number, blocked: number): string {
  const renames =
    ticked === rename ? filesWords(rename) : `${grouped.format(ticked)} of ${filesWords(rename)}`;
  const stuck = blocked === 0 ? [] : [`${grouped.format(blocked)} can't be renamed`];
  return [...(rename === 0 ? [] : [renames]), ...stuck].join(", ");
}

export const companionWords = (companions: readonly string[]) =>
  `Moves with it: ${listed.format(companions)}`;

export const groupWords = "It trades names with another file, so they're renamed or left together.";

export const heldWords =
  "It trades names with a file that can't be renamed now, so it stays where it is.";

/** Why a file stays, naming the other Work of a clash by its title. */
export function blockWords(
  block: FilesRowBlock,
  workId: string,
  other: string | undefined,
): string {
  if (block.reason !== "clash" || block.otherWorkId === undefined) {
    return block.detail;
  }
  if (block.otherWorkId === workId) {
    return "Another file of this Work gets the same new path. Turn on renaming so each file gets its own name.";
  }
  return `${other ?? "Another Work"} has a file with the same new path. Add an id to the folder template so each Work gets its own folder.`;
}
