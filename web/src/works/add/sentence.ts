import type {
  RootsRoot,
  TargetsProfile,
  WorksContentType,
  WorksFieldValue,
  WorksPresetCounts,
} from "../../api/gen/model";
import { type Noun, countWords } from "../../format";
import { yearOf } from "../fields";
import type { AddChoices, MonitorControl, Preset } from "./choices";

const targetNoun = { one: "Target", many: "Targets" } as const;

/** The Work's label with its year, such as "Heat (1995)". */
function labelled(label: string, fields: WorksFieldValue[]) {
  const year = yearOf(fields);
  return year === undefined ? label : `${label} (${year})`;
}

const possessive = (name: string): string => (name.endsWith("s") ? `${name}'` : `${name}'s`);

function wanting(profile: TargetsProfile): string {
  return profile.targets.length === 0
    ? `wanting nothing, because ${profile.name} has no Targets`
    : `wanting ${possessive(profile.name)} ${countWords(profile.targets.length, targetNoun)}`;
}

export const lowered = ({ one, many }: Noun): Noun => ({
  one: one.toLowerCase(),
  many: many.toLowerCase(),
});

function countOf(counts: WorksPresetCounts, preset: Preset) {
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
  if (monitors === 0) {
    return monitorsNew
      ? `monitoring only new ${items.many} as they air`
      : `monitoring no ${items.many}`;
  }
  const every = monitoringOf(counts, { ...choices, preset: "all" }).monitors;
  const all = monitors === every && monitors > 1 ? "all " : "";
  return `monitoring ${all}${countWords(monitors, items)} and ${monitorsNew ? "new ones as they air" : "no new ones"}`;
}

/** The monitoring clause, or none while a choice has no counts because the Work is in the library. */
function monitoringClause(
  type: WorksContentType,
  monitor: MonitorControl,
  counts: WorksPresetCounts | null,
  choices: AddChoices,
): string[] {
  if (monitor.kind === "switch") {
    return [choices.preset === "none" ? "not monitored" : "monitored"];
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
    `Adds ${labelled(work.label, work.fields)} to ${type.name} ${root === undefined ? "with no library root" : `on ${root.path}`}`,
    ...monitoringClause(type, monitor, counts, choices),
    `${wanting(profile)}.`,
  ].join(", ");
}
