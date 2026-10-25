import type { TargetsImpact } from "../api/gen/model";
import { bytes } from "../format";
import type { Unwanted } from "./use-impact-save";

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
        ? "1 download in progress is for a cell that is no longer wanted. Its file is deleted or kept, as chosen below."
        : `${String(downloading)} downloads in progress are for cells that are no longer wanted. Their files are deleted or kept, as chosen below.`),
  ].filter((line) => line !== false);
}

export function saveLabel(action: string, impact: TargetsImpact, unwanted: Unwanted): string {
  const count = impact.unwanted.files;
  return count === 0 ? action : `${action} and ${unwanted} ${counted(count, "file", "files")}`;
}
