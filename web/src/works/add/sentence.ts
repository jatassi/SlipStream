import type {
  RootsRoot,
  TargetsProfile,
  WorksContentType,
  WorksFieldValue,
} from "../../api/gen/model";
import { countWords } from "../../format";
import { fieldText } from "../fields";
import type { AddChoices } from "./choices";

const targetNoun = { one: "Target", many: "Targets" } as const;

export function factTexts(type: WorksContentType, fields: readonly WorksFieldValue[]): string[] {
  return type.fields
    .filter((spec) => spec.tier === "work" && spec.in.includes("facts"))
    .flatMap((spec) => {
      const value = fields.find((field) => field.key === spec.key);
      return value === undefined ? [] : [fieldText(value)];
    });
}

/** The Work's label with its first fact, such as "Heat (1995)". */
export function labelled(
  type: WorksContentType,
  label: string,
  fields: readonly WorksFieldValue[],
) {
  const [first] = factTexts(type, fields);
  return first === undefined ? label : `${label} (${first})`;
}

const possessive = (name: string): string => (name.endsWith("s") ? `${name}'` : `${name}'s`);

function wanting(profile: TargetsProfile): string {
  return profile.targets.length === 0
    ? `wanting nothing, because ${profile.name} has no Targets`
    : `wanting ${possessive(profile.name)} ${countWords(profile.targets.length, targetNoun)}`;
}

export function addSentence(
  work: { readonly label: string; readonly fields: readonly WorksFieldValue[] },
  choices: AddChoices,
  {
    type,
    profiles,
    roots,
  }: {
    type: WorksContentType;
    profiles: readonly TargetsProfile[];
    roots: readonly RootsRoot[];
  },
): string {
  const profile = profiles.find((each) => each.id === choices.profileId);
  if (profile === undefined) {
    throw new Error(`no Profile ${choices.profileId}`);
  }
  const root = roots.find((each) => each.id === choices.rootId);
  return [
    `Adds ${labelled(type, work.label, work.fields)} to ${type.name} ${root === undefined ? "with no library root" : `on ${root.path}`}`,
    choices.monitor ? "monitored" : "not monitored",
    `${wanting(profile)}.`,
  ].join(", ");
}
