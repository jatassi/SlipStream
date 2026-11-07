import type {
  FilesFailedNeed,
  FilesUpgradeNeed,
  WorksContentType,
  WorksItem,
  WorksWorkView,
} from "../../api/gen/model";
import { type FailedGrab, trackStage } from "../../grabs/stages";
import { stageLabel } from "../../ui/stage-track";
import { codeOf } from "../fields-by-tier";
import { anyCut } from "./matrix-model";

/** What a Needs you entry needs to name its Items, Editions and Targets. */
export type Naming = {
  readonly type: WorksContentType;
  readonly work: WorksWorkView;
  readonly targetName: (id: string) => string;
  readonly cutName: (key: string) => string;
};

const list = new Intl.ListFormat("en", { type: "conjunction" });

// "Director's Cut 2160p HDR", led by the Items' codes for a many-Item Work.
function cellsOf(
  { type, work, targetName, cutName }: Naming,
  need: { readonly itemIds: readonly string[]; readonly edition: string },
  targets: readonly string[],
): string {
  const ids = new Set(need.itemIds);
  const items = work.items.filter((item) => ids.has(item.id));
  const first = items.at(0);
  const last = items.at(-1);
  const itemPart =
    type.items === "one" || first === undefined || last === undefined
      ? ""
      : first === last
        ? `${codeOf(type, first)} `
        : `${codeOf(type, first)} to ${codeOf(type, last)} `;
  const cutPart = need.edition === anyCut ? "" : `${cutName(need.edition)} `;
  return `${itemPart}${cutPart}${list.format(targets.map((id) => targetName(id)))}`;
}

export type FailedEntry = { readonly need: FilesFailedNeed; readonly grab: FailedGrab };

const sameTargets = (a: readonly string[], b: readonly string[]) =>
  a.length === b.length && a.every((id, index) => id === b[index]);

/**
 * Adjacent failures of consecutive Items with the same Edition, Target,
 * waiting Targets and failed stage, as one run that keeps the first Grab.
 * The server lists each Grab's failure apart.
 */
export function mergedFailures(
  items: readonly WorksItem[],
  failed: readonly FailedEntry[],
): FailedEntry[] {
  const at = new Map(items.map((item, index) => [item.id, index]));
  const follows = (before: readonly string[], after: readonly string[]) => {
    const last = at.get(before.at(-1) ?? "");
    const next = at.get(after[0] ?? "");
    return last !== undefined && next === last + 1;
  };
  const runs: FailedEntry[] = [];
  for (const entry of failed) {
    const run = runs.at(-1);
    const joins =
      run !== undefined &&
      run.need.edition === entry.need.edition &&
      run.need.targetId === entry.need.targetId &&
      sameTargets(run.need.waiting, entry.need.waiting) &&
      run.grab.failure.stage === entry.grab.failure.stage &&
      follows(run.need.itemIds, entry.need.itemIds);
    if (joins) {
      runs[runs.length - 1] = {
        ...run,
        need: { ...run.need, itemIds: [...run.need.itemIds, ...entry.need.itemIds] },
      };
    } else {
      runs.push(entry);
    }
  }
  return runs;
}

/** A failure's key: its Items, Target and Edition. Grab ids can repeat across a run's Items. */
export const failedKey = ({ need }: FailedEntry) =>
  `${need.itemIds.join()} ${need.targetId} ${need.edition}`;

/** "2160p HDR failed at Import · 1080p and 720p wait on it" */
export function failedWords(naming: Naming, need: FilesFailedNeed, grab: FailedGrab): string {
  const failed = `${cellsOf(naming, need, [need.targetId])} failed at ${stageLabel[trackStage[grab.failure.stage]]}`;
  if (need.waiting.length === 0) {
    return failed;
  }
  const waiting = list.format(need.waiting.map((id) => naming.targetName(id)));
  return `${failed} · ${waiting} ${need.waiting.length === 1 ? "waits" : "wait"} on it`;
}

export const upgradeWords = (naming: Naming, need: FilesUpgradeNeed): string =>
  `Upgrade available for ${cellsOf(naming, need, need.upgrades)}`;
