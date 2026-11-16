import type {
  RootsRoot,
  TargetsProfile,
  WorksContentType,
  WorksFieldValue,
  WorksPresetCounts,
} from "../../api/gen/model";
import { countWords } from "../../format";
import { yearOf } from "../fields";
import { type Preset, lowered, releaseVerb } from "../monitoring/labels";
import type { AddChoices, MonitorControl } from "./choices";

const targetNoun = { one: "Target", many: "Targets" } as const;

/** The Work's label with its year, such as "Heat (1995)". */
function labelled(label: string, fields: WorksFieldValue[]) {
  const year = yearOf(fields);
  return year === undefined ? label : `${label} (${year})`;
}

function targetsSentence(profile: TargetsProfile): string {
  return profile.targets.length === 0
    ? `Its Profile, ${profile.name}, has no Targets, so SlipStream won't download anything for it.`
    : `Its Profile, ${profile.name}, has ${countWords(profile.targets.length, targetNoun)}.`;
}

export function countOf(counts: WorksPresetCounts, preset: Preset) {
  const choice = counts.choices.find((each) => each.preset === preset);
  if (choice === undefined) {
    throw new Error(`the preview counts no ${preset} preset`);
  }
  return choice;
}

/** What a preset monitors now, with Include adding every aside Item, and whether it monitors new Items. */
export function monitoringOf(counts: WorksPresetCounts, choices: AddChoices) {
  const choice = countOf(counts, choices.preset);
  return {
    monitors: choice.monitors + (choices.asides === true ? counts.asides : 0),
    monitorsNew: choice.monitorsNew,
  };
}

function monitoringWords(
  type: WorksContentType,
  counts: WorksPresetCounts,
  choices: AddChoices,
): string {
  const items = lowered(type.item);
  const { monitors, monitorsNew } = monitoringOf(counts, choices);
  const asThey = `as they ${releaseVerb(type)}`;
  if (monitors === 0) {
    return monitorsNew
      ? `SlipStream monitors only new ${items.many} ${asThey}.`
      : `SlipStream monitors no ${items.many}.`;
  }
  const every = monitoringOf(counts, { ...choices, preset: "all" }).monitors;
  const all = monitors === every && monitors > 1 ? "all " : "";
  return `SlipStream monitors ${all}${countWords(monitors, items)} and ${monitorsNew ? `new ones ${asThey}` : "no new ones"}.`;
}

/** The monitoring sentence, or none while a choice has no counts because the Work is in the library. */
function monitoringClause(
  type: WorksContentType,
  monitor: MonitorControl,
  counts: WorksPresetCounts | null,
  choices: AddChoices,
): string[] {
  if (monitor.kind === "switch") {
    return [choices.preset === "none" ? "It isn't monitored." : "SlipStream monitors it."];
  }
  return counts === null ? [] : [monitoringWords(type, counts, choices)];
}

export function addSentence(
  work: { readonly label: string; readonly fields: WorksFieldValue[] },
  choices: AddChoices,
  {
    type,
    profiles,
    roots,
    monitor,
    counts,
  }: {
    type: WorksContentType;
    profiles: readonly TargetsProfile[];
    roots: readonly RootsRoot[];
    monitor: MonitorControl;
    counts: WorksPresetCounts | null;
  },
): string {
  const profile = profiles.find((each) => each.id === choices.profileId);
  if (profile === undefined) {
    throw new Error(`no Profile ${choices.profileId}`);
  }
  const root = roots.find((each) => each.id === choices.rootId);
  return [
    `Adds ${labelled(work.label, work.fields)} to ${type.name} ${root === undefined ? "with no library root" : `on ${root.path}`}.`,
    ...monitoringClause(type, monitor, counts, choices),
    targetsSentence(profile),
  ].join(" ");
}
