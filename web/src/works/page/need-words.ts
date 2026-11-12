import type {
  FilesClientError,
  FilesFailedNeed,
  FilesSpace,
  FilesUpgradeNeed,
  WorksContentType,
  WorksWorkView,
} from "../../api/gen/model";
import { bytes, clock } from "../../format";
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

export type FailedEntry = {
  readonly need: FilesFailedNeed;
  readonly grab: FailedGrab;
};

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

export const clientErrorWords = ({ client, node, message, failsAt }: FilesClientError) => {
  const said = /[.!?]$/u.test(message) ? message : `${message}.`;
  return `${client} on ${node} reports "${said}" Fix it in ${client}. SlipStream fails this Grab if the error lasts past ${clock(failsAt)}.`;
};

/** The words of an Import's wait, which Pending imports shows in full. */
export type ImportWait = "no_space" | "root_not_ready" | "probe_timeout";

/**
 * An Import's wait, naming its library root as root words name it and, for
 * no_space, the bytes free and needed.
 */
export function importWaitWords(code: ImportWait, root: string, space: FilesSpace | null): string {
  switch (code) {
    case "no_space":
      return `${
        space === null
          ? `The ${root} has too little free space for this file.`
          : `The ${root} has ${bytes(space.free)} free above its error level, and this file needs ${bytes(space.needs)}.`
      } Free up space there. SlipStream retries automatically once there's room.`;
    case "root_not_ready":
      return `The ${root} isn't ready. SlipStream retries once it is.`;
    case "probe_timeout":
      return "Reading a file of this download timed out 3 times. Retry, or check the Node's disk.";
  }
}
