import type {
  WorksContentType,
  WorksMonitoringChange,
  WorksMonitoringList,
  WorksPresetCount,
  WorksPresetCounts,
} from "../../api/gen/model";
import { countWords } from "../../format";
import { type Preset, lowered } from "./labels";

/** A preset for one or many Works, with its two rules. */
export type MonitoringChoice = {
  readonly preset: Preset;
  readonly includeAsides: boolean;
  /** Undefined follows the preset. */
  readonly monitorNew: boolean | undefined;
};

export const monitoringBody = (
  workIds: readonly string[],
  { preset, includeAsides, monitorNew }: MonitoringChoice,
  preview: boolean,
): WorksMonitoringChange => ({
  workIds: [...workIds],
  preset,
  includeAsides,
  ...(monitorNew === undefined ? {} : { monitorNew }),
  preview,
});

/** What a choice would do across the Works: each preset's counts summed, and the Items it moves. */
export type MonitoringPreview = {
  readonly counts: WorksPresetCounts;
  readonly start: number;
  readonly stop: number;
};

function summed(each: readonly WorksPresetCounts[]): WorksPresetCounts {
  const choices = new Map<Preset, WorksPresetCount>();
  for (const counts of each) {
    for (const choice of counts.choices) {
      const held = choices.get(choice.preset);
      choices.set(choice.preset, { ...choice, monitors: (held?.monitors ?? 0) + choice.monitors });
    }
  }
  return {
    asides: each.reduce((total, counts) => total + counts.asides, 0),
    choices: [...choices.values()],
  };
}

export function previewOf({ outcomes }: WorksMonitoringList): MonitoringPreview {
  const counts = outcomes.flatMap((outcome) =>
    outcome.outcome !== "failed" && outcome.counts !== null ? [outcome.counts] : [],
  );
  const done = outcomes.flatMap((outcome) => (outcome.outcome === "done" ? [outcome] : []));
  return {
    counts: summed(counts),
    start: done.reduce((total, { start }) => total + start, 0),
    stop: done.reduce((total, { stop }) => total + stop, 0),
  };
}

/** What a preset monitors now, Include adding every aside Item, and whether it monitors new Items. */
export function countOf(counts: WorksPresetCounts, preset: Preset, includeAsides: boolean) {
  const choice = counts.choices.find((each) => each.preset === preset);
  if (choice === undefined) {
    throw new Error(`the preview counts no ${preset} preset`);
  }
  return {
    monitors: choice.monitors + (includeAsides ? counts.asides : 0),
    monitorsNew: choice.monitorsNew,
  };
}

const agreed = (count: number, verb: string) => (count === 1 ? `${verb}s` : verb);

/** "6 stories start being monitored and 2 stop.", across the Works when there are several. */
export function changeWords(
  type: WorksContentType,
  { start, stop }: MonitoringPreview,
  works: number,
): string {
  const items = lowered(type.item);
  const across = works > 1 ? ` across ${countWords(works, lowered(type.work))}` : "";
  if (start === 0 && stop === 0) {
    return `No ${items.many} change${across}.`;
  }
  const starting = `${countWords(start, items)} ${agreed(start, "start")} being monitored`;
  const stopping = agreed(stop, "stop");
  if (stop === 0) {
    return `${starting}${across}.`;
  }
  if (start === 0) {
    return `${countWords(stop, items)} ${stopping} being monitored${across}.`;
  }
  return `${starting} and ${stop} ${stopping}${across}.`;
}
