import type {
  NamingTypeNaming,
  SettingChoiceField,
  SettingSwitchField,
  SettingTemplateField,
} from "../api/gen/model";

/** A Content Type's naming settings as its card lays them out. */
export type NamingCard = {
  readonly rename: SettingSwitchField;
  /** The folder templates outermost first, then the file templates. */
  readonly templates: readonly SettingTemplateField[];
  readonly colon: SettingChoiceField;
  /** The text the Custom colon mode puts in a colon's place. */
  readonly colonCustom: SettingTemplateField;
  /** Left out for a type whose file names number no Items. */
  readonly multiEpisode: SettingChoiceField | undefined;
};

export const customColon = "custom";

export function cardOf({ contentType, settings }: NamingTypeNaming): NamingCard {
  const named = (field: string) =>
    settings.find((setting) => setting.key === `${contentType}.naming.${field}`);
  const rename = named("renameFiles");
  const colon = named("colon");
  const colonCustom = named("colonCustom");
  const multiEpisode = named("multiEpisode");
  if (rename?.kind !== "switch" || colon?.kind !== "choice" || colonCustom?.kind !== "template") {
    throw new Error(`the naming settings of ${contentType} lack the rename switch or colon mode`);
  }
  return {
    rename,
    templates: settings.filter(
      (setting): setting is SettingTemplateField => setting.kind === "template" && !setting.plain,
    ),
    colon,
    colonCustom,
    multiEpisode: multiEpisode?.kind === "choice" ? multiEpisode : undefined,
  };
}

/** A value as a save sends it: null puts back the recommended default. */
export type DraftValue = string | boolean | null;

/** The card's unsaved values by setting key. A key holds only while its value differs from the saved one. */
export type Draft = ReadonlyMap<string, DraftValue>;

type Field = SettingSwitchField | SettingTemplateField | SettingChoiceField;

export function edit(draft: Draft, field: Field, value: DraftValue): Draft {
  const next = new Map(draft);
  const unchanged = value === null ? field.isDefault : value === field.value;
  if (unchanged) {
    next.delete(field.key);
  } else {
    next.set(field.key, value);
  }
  return next;
}

export function textOf(field: SettingTemplateField | SettingChoiceField, draft: Draft): string {
  const value = draft.get(field.key);
  if (value === null) {
    return field.default;
  }
  return typeof value === "string" ? value : field.value;
}

export function switchOf(field: SettingSwitchField, draft: Draft): boolean {
  const value = draft.get(field.key);
  if (value === null) {
    return field.default;
  }
  return typeof value === "boolean" ? value : field.value;
}

/**
 * The values that put back the recommended templates: every template that
 * differs from its default, and every setting the save check refuses.
 */
export function recommendedValues(
  card: NamingCard,
  problems: NamingTypeNaming["problems"],
): Record<string, null> {
  const keys = [
    ...card.templates.filter((field) => !field.isDefault).map((field) => field.key),
    ...problems.map((problem) => problem.key),
  ];
  return Object.fromEntries(keys.map((key) => [key, null]));
}
