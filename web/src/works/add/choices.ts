import type {
  RootsRoot,
  SettingField,
  TargetsProfile,
  WorksContentType,
} from "../../api/gen/model";

/** A Content Type's three add defaults, one stored value each. */
export type AddDefaults = {
  readonly profileId: string;
  /** Undefined while the type has no root. */
  readonly rootId: string | undefined;
  readonly monitor: boolean;
};

export type AddChoices = AddDefaults & { readonly remember: boolean };

const defaultKinds = ["profile", "root", "monitor"] as const;

export type Default = (typeof defaultKinds)[number];

export const addMonitorKey = (type: WorksContentType): string => `${type.id}.add.monitor`;

export function addDefaultsOf({
  type,
  profiles,
  roots,
  settings,
}: {
  type: WorksContentType;
  profiles: readonly TargetsProfile[];
  roots: readonly RootsRoot[];
  settings: readonly SettingField[];
}): AddDefaults {
  const profile = profiles.find((each) => each.contentType === type.id && each.default);
  if (profile === undefined) {
    throw new Error(`${type.id} has no default Profile`);
  }
  const monitor = settings.find((field) => field.key === addMonitorKey(type));
  if (monitor?.kind !== "switch") {
    throw new Error(`${type.id} has no ${addMonitorKey(type)} switch`);
  }
  return {
    profileId: profile.id,
    rootId: roots.find((root) => root.contentType === type.id && root.default)?.id,
    monitor: monitor.value,
  };
}

export const startChoices = (defaults: AddDefaults): AddChoices => ({
  ...defaults,
  remember: true,
});

const comparisons: { readonly [D in Default]: (c: AddChoices, d: AddDefaults) => boolean } = {
  profile: (c, d) => c.profileId !== d.profileId,
  root: (c, d) => c.rootId !== d.rootId,
  monitor: (c, d) => c.monitor !== d.monitor,
};

/** The defaults an add with these choices would move while Remember is on. */
export function movedDefaults(choices: AddChoices, defaults: AddDefaults): readonly Default[] {
  if (!choices.remember) {
    return [];
  }
  return defaultKinds.filter((kind) => comparisons[kind](choices, defaults));
}
