import type { TargetsNewWants, TargetsProfile, TargetsWants } from "../api/gen/model";
import { type EditionsDraft, editionsBodyOf, editionsDraftOf } from "../targets/editions-draft";

// `own` is null while the Work takes its Profile's Edition list.
export type WantsDraft = { readonly profileId: string; readonly own: EditionsDraft | null };

export const wantsDraftOf = (wants: TargetsWants): WantsDraft => ({
  profileId: wants.profileId,
  own: wants.from === "work" ? editionsDraftOf(wants.editions) : null,
});

export const editionsOf = (draft: WantsDraft, profile: TargetsProfile): EditionsDraft =>
  draft.own ?? editionsDraftOf(profile.editions);

export const wantsBodyOf = (draft: WantsDraft): TargetsNewWants => ({
  profileId: draft.profileId,
  editions: draft.own === null ? null : editionsBodyOf(draft.own),
});
