import type {
  RootsRoot,
  SettingField,
  TargetsProfile,
  WorksContentType,
} from "../../api/gen/model";
import { type Preset, isPreset } from "../monitoring/labels";

export type PresetOption = { readonly preset: Preset; readonly label: string };

/** How the sheet asks what to monitor, from the kind of the type's add.monitor setting. */
export type MonitorControl =
  | { readonly kind: "switch" }
  | { readonly kind: "choice"; readonly label: string; readonly options: readonly PresetOption[] };

/** A Content Type's add defaults, one stored value each. */
export type AddDefaults = {
  readonly profileId: string;
  /** Undefined while the type has no root. */
  readonly rootId: string | undefined;
  readonly preset: Preset;
  /** Undefined while the type has no add.asides setting. */
  readonly asides: boolean | undefined;
  /** Undefined while the type has no add.levelFolders setting. */
  readonly levelFolders: boolean | undefined;
};

export type AddChoices = AddDefaults & { readonly remember: boolean };

export type AddSetup = { readonly defaults: AddDefaults; readonly monitor: MonitorControl };

const defaultKinds = ["profile", "root", "monitor", "asides", "levelFolders"] as const;

export type Default = (typeof defaultKinds)[number];

const keyOf = (type: WorksContentType, name: string) => `${type.id}.add.${name}`;

function monitorOf(
  type: WorksContentType,
  settings: readonly SettingField[],
): { preset: Preset; control: MonitorControl } {
  const field = settings.find((each) => each.key === keyOf(type, "monitor"));
  if (field?.kind === "switch") {
    return { preset: field.value ? "all" : "none", control: { kind: "switch" } };
  }
  if (field?.kind !== "choice") {
    throw new Error(`${type.id} has no ${keyOf(type, "monitor")} switch or choice`);
  }
  const options = field.options.map(({ value, label }) => {
    if (!isPreset(value)) {
      throw new Error(`${field.key} offers ${value}, which is no preset`);
    }
    return { preset: value, label };
  });
  if (!isPreset(field.value)) {
    throw new Error(`${field.key} holds ${field.value}, which is no preset`);
  }
  return { preset: field.value, control: { kind: "choice", label: field.label, options } };
}

function switchOf(
  type: WorksContentType,
  settings: readonly SettingField[],
  name: string,
): boolean | undefined {
  const field = settings.find((each) => each.key === keyOf(type, name));
  if (field === undefined) {
    return undefined;
  }
  if (field.kind !== "switch") {
    throw new Error(`${field.key} is a ${field.kind}, not a switch`);
  }
  return field.value;
}

export function addSetupOf({
  type,
  profiles,
  roots,
  settings,
}: {
  type: WorksContentType;
  profiles: readonly TargetsProfile[];
  roots: readonly RootsRoot[];
  settings: readonly SettingField[];
}): AddSetup {
  const profile = profiles.find((each) => each.contentType === type.id && each.default);
  if (profile === undefined) {
    throw new Error(`${type.id} has no default Profile`);
  }
  const monitor = monitorOf(type, settings);
  return {
    defaults: {
      profileId: profile.id,
      rootId: roots.find((root) => root.contentType === type.id && root.default)?.id,
      preset: monitor.preset,
      asides: switchOf(type, settings, "asides"),
      levelFolders: switchOf(type, settings, "levelFolders"),
    },
    monitor: monitor.control,
  };
}

export const startChoices = (defaults: AddDefaults): AddChoices => ({
  ...defaults,
  remember: true,
});

const comparisons: { readonly [D in Default]: (c: AddChoices, d: AddDefaults) => boolean } = {
  profile: (c, d) => c.profileId !== d.profileId,
  root: (c, d) => c.rootId !== d.rootId,
  monitor: (c, d) => c.preset !== d.preset,
  asides: (c, d) => c.asides !== d.asides,
  levelFolders: (c, d) => c.levelFolders !== d.levelFolders,
};

/** The defaults an add with these choices would move while Remember is on. */
export function movedDefaults(choices: AddChoices, defaults: AddDefaults): readonly Default[] {
  if (!choices.remember) {
    return [];
  }
  return defaultKinds.filter((kind) => comparisons[kind](choices, defaults));
}
