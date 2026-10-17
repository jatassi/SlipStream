import type { TargetsNewWants, TargetsProfile, TargetsWants } from "../api/gen/model";
import { type EditionsDraft, editionsDraftOf } from "../targets/editions-draft";

/** A Work's wants being edited: its Profile, and its own Edition list or null for the Profile's. */
export type WantsDraft = { readonly profileId: string; readonly own: EditionsDraft | null };

export const wantsDraftOf = (wants: TargetsWants): WantsDraft => ({
  profileId: wants.profileId,
  own: wants.from === "work" ? editionsDraftOf(wants.editions) : null,
});

/** The Edition list the Work wants under the draft: its own, or its Profile's. */
export const editionsOf = (draft: WantsDraft, profile: TargetsProfile): EditionsDraft =>
  draft.own ?? editionsDraftOf(profile.editions);

// Only Any cut is sent: a chosen cut list is saved once the API stores one.
export function wantsBodyOf(draft: WantsDraft): TargetsNewWants {
  if (draft.own?.kind === "cuts") {
    throw new Error("Chosen cuts can't be saved yet. Choose Any cut to save this Work's Editions.");
  }
  return {
    profileId: draft.profileId,
    editions: draft.own === null ? null : { kind: draft.own.kind },
  };
}
