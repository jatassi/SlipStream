import type { TargetsNewWants, TargetsProfile, TargetsWants } from "../api/gen/model";
import { type EditionsDraft, editionsBodyOf, editionsDraftOf } from "../targets/editions-draft";

/** A Work's wants being edited: its Profile, and its own Edition list or null for the Profile's. */
export type WantsDraft = { readonly profileId: string; readonly own: EditionsDraft | null };

export const wantsDraftOf = (wants: TargetsWants): WantsDraft => ({
  profileId: wants.profileId,
  own: wants.from === "work" ? editionsDraftOf(wants.editions) : null,
});

/** The Edition list the Work wants under the draft: its own, or its Profile's. */
export const editionsOf = (draft: WantsDraft, profile: TargetsProfile): EditionsDraft =>
  draft.own ?? editionsDraftOf(profile.editions);

export const wantsBodyOf = (draft: WantsDraft): TargetsNewWants => ({
  profileId: draft.profileId,
  editions: draft.own === null ? null : editionsBodyOf(draft.own),
});
