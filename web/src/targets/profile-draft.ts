import type {
  TargetsEntry,
  TargetsEntrySourcing,
  TargetsNewProfile,
  TargetsProfile,
  TargetsWantedLanguage,
} from "../api/gen/model";
import { duplicateWords } from "../subtitles/settings-words";
import { type EditionsDraft, editionsBodyOf, editionsDraftOf } from "./editions-draft";

export type GraceUnit = "hours" | "days";

/** How long a "Downloaded, or made from" row waits for a download, as typed. */
export type GraceDraft = { readonly amount: string; readonly unit: GraceUnit };

export type Made = Exclude<TargetsEntrySourcing, "download">;

/** A Target's row: how its file is sourced and, when it is made, from which Target above it. */
export type EntryDraft = {
  readonly targetId: string;
  /** Kept while the row is downloaded, so choosing a wait again finds what was typed. */
  readonly grace: GraceDraft;
} & ({ readonly sourcing: "download" } | { readonly sourcing: Made; readonly from: string });

const graceUnitSeconds: Readonly<Record<GraceUnit, number>> = { hours: 3600, days: 86_400 };

const shortestGrace = graceUnitSeconds.hours;
const longestGrace = 90 * graceUnitSeconds.days;

export const defaultGrace: GraceDraft = { amount: "3", unit: "days" };

export function graceDraftOf(seconds: number): GraceDraft {
  const unit: GraceUnit = seconds % graceUnitSeconds.days === 0 ? "days" : "hours";
  return { amount: String(seconds / graceUnitSeconds[unit]), unit };
}

export const graceSecondsOf = ({ amount, unit }: GraceDraft): number =>
  Number(amount) * graceUnitSeconds[unit];

export function graceProblem(grace: GraceDraft): string | undefined {
  const seconds = graceSecondsOf(grace);
  if (grace.amount.trim() === "" || !(seconds >= shortestGrace && seconds <= longestGrace)) {
    return "Between 1 hour and 90 days.";
  }
  return Number.isInteger(seconds / graceUnitSeconds.hours) ? undefined : "In whole hours.";
}

export const downloaded = (targetId: string): EntryDraft => ({
  targetId,
  sourcing: "download",
  grace: defaultGrace,
});

function entryDraftOf({
  targetId,
  sourcing,
  fromTargetId,
  graceSeconds,
}: TargetsEntry): EntryDraft {
  const grace = graceSeconds === undefined ? defaultGrace : graceDraftOf(graceSeconds);
  return sourcing === "download" || fromTargetId === undefined
    ? { targetId, sourcing: "download", grace }
    : { targetId, sourcing, from: fromTargetId, grace };
}

/** The Targets above `index` whose files are downloaded, which a made row can be made from. */
export const sourcesAbove = (entries: readonly EntryDraft[], index: number): string[] =>
  entries.slice(0, index).flatMap((entry) => (entry.sourcing === "derive" ? [] : [entry.targetId]));

/** Whether a made row's source is no longer a downloaded Target above it. */
export const sourceLost = (entries: readonly EntryDraft[], index: number): boolean => {
  const entry = entries[index];
  return (
    entry !== undefined &&
    entry.sourcing !== "download" &&
    !sourcesAbove(entries, index).includes(entry.from)
  );
};

/**
 * A Profile being edited. `id` is null until it is created; once set, the
 * Content Type is fixed and the Profile's estimates can be read.
 */
export type ProfileDraft = {
  readonly id: string | null;
  readonly name: string;
  readonly contentType: string;
  /** Its Target rows, in Profile order. */
  readonly targets: readonly EntryDraft[];
  readonly editions: EditionsDraft;
  readonly languages: readonly TargetsWantedLanguage[];
};

export const newProfileDraft = (contentType: string): ProfileDraft => ({
  id: null,
  name: "",
  contentType,
  targets: [],
  editions: editionsDraftOf({ kind: "any" }),
  languages: [],
});

export const profileDraftOf = (profile: TargetsProfile): ProfileDraft => ({
  id: profile.id,
  name: profile.name,
  contentType: profile.contentType,
  targets: profile.targets.map(entryDraftOf),
  editions: editionsDraftOf(profile.editions),
  languages: profile.languages,
});

/** The index of the first Wanted Language that repeats an earlier one's language and kind. */
export function duplicateOf(languages: readonly TargetsWantedLanguage[]): number | undefined {
  const at = languages.findIndex((want, index) =>
    languages
      .slice(0, index)
      .some((earlier) => earlier.language === want.language && earlier.forced === want.forced),
  );
  return at === -1 ? undefined : at;
}

export function profileProblemsOf(draft: ProfileDraft): string[] {
  const duplicate = duplicateOf(draft.languages);
  const twice = duplicate === undefined ? undefined : draft.languages[duplicate];
  const sourcing = draft.targets.some(
    (entry, index) =>
      sourceLost(draft.targets, index) ||
      (entry.sourcing === "download_then_derive" && graceProblem(entry.grace) !== undefined),
  );
  return [
    ...(draft.name.trim() === "" ? ["Name the Profile."] : []),
    ...(sourcing ? ["Fix the marked Target rows."] : []),
    ...(twice === undefined ? [] : [duplicateWords(twice)]),
  ];
}

function entryBodyOf(entry: EntryDraft): TargetsEntry {
  const { targetId } = entry;
  if (entry.sourcing === "download") {
    return { targetId, sourcing: "download" };
  }
  return entry.sourcing === "derive"
    ? { targetId, sourcing: "derive", fromTargetId: entry.from }
    : {
        targetId,
        sourcing: "download_then_derive",
        fromTargetId: entry.from,
        graceSeconds: graceSecondsOf(entry.grace),
      };
}

export const profileBodyOf = (draft: ProfileDraft): TargetsNewProfile => ({
  name: draft.name.trim(),
  contentType: draft.contentType,
  targets: draft.targets.map(entryBodyOf),
  editions: editionsBodyOf(draft.editions),
  languages: [...draft.languages],
});
