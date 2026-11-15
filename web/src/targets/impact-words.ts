import type { TargetsImpact } from "../api/gen/model";
import { bytes } from "../format";
import type { Unwanted } from "./use-impact-save";

const grouped = new Intl.NumberFormat("en");

const counted = (count: number, one: string, many: string) =>
  `${String(count)} ${count === 1 ? one : many}`;

export function impactLines(impact: TargetsImpact): string[] {
  const { works, ownList, unwanted, renamed, newlyWanted, downloading } = impact;
  return [
    works > 0 && `${counted(works, "Work changes", "Works change")}.`,
    ownList > 0 &&
      `${counted(ownList, "Work keeps its own Edition list", "Works keep their own Edition lists")}, so the new list doesn't apply to ${ownList === 1 ? "it" : "them"}.`,
    unwanted.files > 0 &&
      `${counted(unwanted.files, "file would no longer be used by any Target", "files would no longer be used by any Target")} (${bytes(unwanted.bytes)}).`,
    renamed > 0 && `${counted(renamed, "file is renamed", "files are renamed")}.`,
    newlyWanted > 0 &&
      `SlipStream will look for ${counted(newlyWanted, "more file", "more files")}.`,
    downloading > 0 &&
      (downloading === 1
        ? "1 download in progress is no longer needed. When it finishes, it waits in Pending imports."
        : `${String(downloading)} downloads in progress are no longer needed. When they finish, they wait in Pending imports.`),
  ].filter((line) => line !== false);
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
  return count === 0 ? action : `${action} and ${unwanted} ${counted(count, "file", "files")}`;
}
