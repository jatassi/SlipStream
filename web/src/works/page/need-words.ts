import type {
  FilesFailedNeed,
  FilesUpgradeNeed,
  WorksContentType,
  WorksWorkView,
} from "../../api/gen/model";
import { type FailedGrab, trackStage } from "../../grabs/stages";
import { stageLabel } from "../../ui/stage-track";
import { anyCut } from "./matrix-model";

/** What a Needs you entry needs to name its Items, Editions and Targets. */
export type Naming = {
  readonly type: WorksContentType;
  readonly work: WorksWorkView;
  readonly targetName: (id: string) => string;
  readonly cutName: (key: string) => string;
};

const list = new Intl.ListFormat("en", { type: "conjunction" });

// "Director's Cut 2160p HDR", led by the Items for a many-Item Work.
function cellsOf(
  { type, work, targetName, cutName }: Naming,
  need: { readonly itemIds: readonly string[]; readonly edition: string },
  targets: readonly string[],
): string {
  const items = work.items.filter((item) => need.itemIds.includes(item.id));
  const first = items.at(0);
  const last = items.at(-1);
  const itemPart =
    type.items === "one" || first === undefined || last === undefined
      ? ""
      : first === last
        ? `${first.label} `
        : `${first.label} to ${last.label} `;
  const cutPart = need.edition === anyCut ? "" : `${cutName(need.edition)} `;
  return `${itemPart}${cutPart}${list.format(targets.map((id) => targetName(id)))}`;
}

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
