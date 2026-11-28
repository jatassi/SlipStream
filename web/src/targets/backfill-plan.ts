import type { TargetsBackfill, TargetsImpact, TargetsProfile } from "../api/gen/model";
import { useListTargetsSuspense } from "../api/gen/query/targets/targets";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { type Noun, bytes, countWords, spanWords } from "../format";
import { worksNounOf } from "../wanted/wanted-words";
import type { ProfileDraft } from "./profile-draft";
import { listed } from "./words";

/** A Profile save that makes files for Works it already has, waiting on the Admin's choices. */
export type Backfilling = {
  readonly draft: ProfileDraft;
  readonly impact: TargetsImpact;
  /** The Profiles of its Content Type, it among them. */
  readonly profiles: readonly TargetsProfile[];
  /** The save found more to make than the preview before it showed. */
  readonly changed: boolean;
};

/** What the dialog names, worked out once from the saved Profile and the draft. */
export type Shown = {
  readonly saved: TargetsProfile;
  readonly others: readonly TargetsProfile[];
  readonly savedDefault: string;
  /** The draft's name for the Profile. */
  readonly profile: string;
  /** The Targets the save starts making, listed. */
  readonly made: string;
  readonly noun: Noun;
  readonly targetName: (id: string) => string;
  readonly profileName: (id: string) => string;
};

export type Convert = "all" | "choose" | "none";

export type Plan = {
  readonly convert: Convert;
  readonly picked: ReadonlySet<string>;
  /** "new", or the id of the Profile the Works left out move to. */
  readonly rest: string;
  readonly newName: string;
  readonly defaultId: string;
};

const capital = (word: string) => `${word.charAt(0).toUpperCase()}${word.slice(1)}`;

export const stepsOf = (noun: Noun) => ["Cost", capital(noun.many), "The rest"] as const;

const sameWants = (a: TargetsProfile, b: TargetsProfile) =>
  JSON.stringify([a.targets, a.editions, a.languages]) ===
  JSON.stringify([b.targets, b.editions, b.languages]);

/** The Targets the draft makes that the saved Profile didn't. */
export const newlyMade = (saved: TargetsProfile, draft: ProfileDraft): string[] =>
  draft.targets.flatMap((entry) => {
    const before = saved.targets.find((old) => old.targetId === entry.targetId);
    const was = before !== undefined && before.sourcing !== "download";
    return entry.sourcing !== "download" && !was ? [entry.targetId] : [];
  });

/** Converts every Work, and sends any left out to a Profile that wants what the old one did, when one exists. */
export function firstPlan(shown: Shown): Plan {
  const matching = shown.others.find((other) => sameWants(other, shown.saved));
  return {
    convert: "all",
    picked: new Set(),
    rest: matching?.id ?? "new",
    newName: `${shown.profile} (without ${shown.made})`,
    defaultId: shown.savedDefault,
  };
}

export function convertedOf(plan: Plan, impact: TargetsImpact) {
  const rows = impact.made.rows.filter(
    (row) => plan.convert === "all" || (plan.convert === "choose" && plan.picked.has(row.workId)),
  );
  return {
    ids: rows.map((row) => row.workId),
    files: rows.reduce((sum, row) => sum + row.files, 0),
    seconds: rows.reduce((sum, row) => sum + row.seconds, 0),
    bytes: rows.reduce((sum, row) => sum + row.bytes, 0),
    rest: impact.made.rows.length - rows.length,
  };
}

/** Whether the save can go: the Works left out need a name for the new Profile they move to. */
export const unnamed = (plan: Plan, impact: TargetsImpact): boolean =>
  convertedOf(plan, impact).rest > 0 && plan.rest === "new" && plan.newName.trim() === "";

/** The sentence that restates the whole save before the Admin confirms it. */
export function restating(plan: Plan, impact: TargetsImpact, shown: Shown): string {
  const converted = convertedOf(plan, impact);
  const restName = plan.rest === "new" ? plan.newName.trim() : shown.profileName(plan.rest);
  const defaultName = shown.profileName(plan.defaultId);
  const stays = plan.defaultId === shown.savedDefault ? "stays" : "becomes";
  return [
    `Save ${shown.profile} with ${shown.made}.`,
    converted.ids.length > 0 &&
      `Convert ${countWords(converted.ids.length, shown.noun)} after every other Conversion, which takes about ${spanWords(converted.seconds)} and ${bytes(converted.bytes)}.`,
    converted.rest > 0 && `Move ${String(converted.rest)} to ${restName}.`,
    `${defaultName} ${stays} the default for new ${shown.noun.many}.`,
  ]
    .filter((part) => part !== false)
    .join(" ");
}

export const backfillOf = (plan: Plan, impact: TargetsImpact): TargetsBackfill => ({
  convert: convertedOf(plan, impact).ids,
  rest: plan.rest === "new" ? { newName: plan.newName.trim() } : { profileId: plan.rest },
  default: plan.defaultId,
});

/** The warning for moving Works to a Profile that lacks some of the saved one's Targets. */
export function lackingWords(shown: Shown, other: TargetsProfile): string | undefined {
  const missing = shown.saved.targets
    .filter((entry) => !other.targets.some((has) => has.targetId === entry.targetId))
    .map((entry) => shown.targetName(entry.targetId));
  return missing.length === 0 ? undefined : `${other.name} doesn't have ${listed(missing)}.`;
}

/** What the open dialog names, from the saved Profile, the draft and the library's Targets. */
export function useShown(
  saved: TargetsProfile,
  backfilling: Backfilling | undefined,
): Shown | undefined {
  const { targets } = useListTargetsSuspense().data;
  const types = useListContentTypesSuspense().data.contentTypes;
  if (backfilling === undefined) {
    return undefined;
  }
  const { draft, profiles } = backfilling;
  const targetName = (id: string) => targets.find((target) => target.id === id)?.name ?? id;
  const profile = draft.name.trim();
  const others = profiles.filter((other) => other.id !== saved.id);
  return {
    saved,
    others,
    savedDefault: profiles.find((other) => other.default)?.id ?? saved.id,
    profile,
    made: listed(newlyMade(saved, draft).map((id) => targetName(id))),
    noun: worksNounOf(types.find((type) => type.id === saved.contentType)),
    targetName,
    profileName: (id) =>
      id === saved.id ? profile : (others.find((other) => other.id === id)?.name ?? id),
  };
}
