import type { TargetsNewProfile, TargetsProfile, TargetsWantedLanguage } from "../api/gen/model";
import { duplicateWords } from "../subtitles/settings-words";
import { type EditionsDraft, editionsBodyOf, editionsDraftOf } from "./editions-draft";

/**
 * A Profile being edited. `id` is null until it is created; once set, the
 * Content Type is fixed and the Profile's estimates can be read.
 */
export type ProfileDraft = {
  readonly id: string | null;
  readonly name: string;
  readonly contentType: string;
  /** Target ids, in Profile order. */
  readonly targets: readonly string[];
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
  targets: profile.targets.map((entry) => entry.targetId),
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
  return [
    ...(draft.name.trim() === "" ? ["Name the Profile."] : []),
    ...(twice === undefined ? [] : [duplicateWords(twice)]),
  ];
}

export const profileBodyOf = (draft: ProfileDraft): TargetsNewProfile => ({
  name: draft.name.trim(),
  contentType: draft.contentType,
  targets: draft.targets.map((targetId) => ({ targetId, sourcing: "download" })),
  editions: editionsBodyOf(draft.editions),
  languages: [...draft.languages],
});
