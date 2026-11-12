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
      `${counted(unwanted.files, "file would fill nothing", "files would fill nothing")} (${bytes(unwanted.bytes)}).`,
    renamed > 0 && `${counted(renamed, "file is renamed", "files are renamed")}.`,
    newlyWanted > 0 && `${counted(newlyWanted, "cell becomes wanted", "cells become wanted")}.`,
    downloading > 0 &&
      (downloading === 1
        ? "1 download in progress is for cells that are no longer wanted. It waits in Pending imports once it finishes."
        : `${String(downloading)} downloads in progress are for cells that are no longer wanted. They wait in Pending imports once they finish.`),
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
