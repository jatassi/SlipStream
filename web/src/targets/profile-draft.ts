import type { TargetsNewProfile, TargetsProfile } from "../api/gen/model";
import { type EditionsDraft, editionsDraftOf } from "./editions-draft";

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
};

export const newProfileDraft = (contentType: string): ProfileDraft => ({
  id: null,
  name: "",
  contentType,
  targets: [],
  editions: editionsDraftOf({ kind: "any" }),
});

export const profileDraftOf = (profile: TargetsProfile): ProfileDraft => ({
  id: profile.id,
  name: profile.name,
  contentType: profile.contentType,
  targets: profile.targets.map((entry) => entry.targetId),
  editions: editionsDraftOf(profile.editions),
});

export const profileProblemsOf = (draft: ProfileDraft): string[] =>
  draft.name.trim() === "" ? ["Name the Profile."] : [];

// Only Any cut is sent: a chosen cut list is saved once the API stores one.
export function profileBodyOf(draft: ProfileDraft): TargetsNewProfile {
  if (draft.editions.kind === "cuts") {
    throw new Error("Chosen cuts can't be saved yet. Choose Any cut to save the Profile.");
  }
  return {
    name: draft.name.trim(),
    contentType: draft.contentType,
    targets: draft.targets.map((targetId) => ({ targetId, sourcing: "download" })),
    editions: { kind: draft.editions.kind },
  };
}
