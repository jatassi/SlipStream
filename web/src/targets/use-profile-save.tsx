import { type ReactElement, useState } from "react";

import type { TargetsBackfill, TargetsNewProfile, TargetsProfile } from "../api/gen/model";
import { listProfiles, updateProfile } from "../api/gen/query/targets/targets";
import { toast } from "../ui/toast";
import { BackfillDialog } from "./backfill-dialog";
import { type Backfilling, useShown } from "./backfill-plan";
import { savingCuts } from "./editions-draft";
import { useImpactSheet } from "./impact-sheet";
import { type ProfileDraft, profileBodyOf } from "./profile-draft";
import { type ImpactChange, runConfirmed } from "./use-impact-save";

const changeOf = (
  profile: TargetsProfile,
  body: TargetsNewProfile,
  onSaved: () => Promise<void> | void,
): ImpactChange => ({
  title: `Save ${profile.name}`,
  action: "Save",
  run: (changing) => savingCuts(() => updateProfile(profile.id, body, changing)),
  done: async () => {
    toast(`Saved ${body.name}.`);
    await onSaved();
  },
});

const previewOf = async (change: ImpactChange) => (await change.run({ preview: true })).impact;

/**
 * Saves a Profile through the backfill dialog when the change makes files
 * for Works it already has and deletes none, and through the Impact sheet
 * otherwise. A save that would make more than was confirmed is previewed
 * again and shown with its fresh counts.
 */
export function useProfileSave(
  profile: TargetsProfile,
  onSaved: () => Promise<void> | void,
): { readonly save: (draft: ProfileDraft) => Promise<void>; readonly dialogs: ReactElement } {
  const sheet = useImpactSheet();
  const [backfilling, setBackfilling] = useState<Backfilling>();
  const shown = useShown(profile, backfilling);
  const changeFor = (draft: ProfileDraft, backfill?: TargetsBackfill) =>
    changeOf(profile, { ...profileBodyOf(draft), ...(backfill && { backfill }) }, onSaved);

  const save = async (draft: ProfileDraft) => {
    const change = changeFor(draft);
    const impact = await previewOf(change);
    if (impact.made.files === 0 || impact.unwanted.files > 0) {
      await sheet.saveFrom(change, impact);
      return;
    }
    const { profiles } = await listProfiles();
    const ofType = profiles.filter((other) => other.contentType === profile.contentType);
    setBackfilling({ draft, impact, profiles: ofType, changed: false });
  };

  const commit = async (open: Backfilling, backfill: TargetsBackfill, made: number) => {
    const change = changeFor(open.draft, backfill);
    if ((await runConfirmed(change, "delete", open.impact, made)) === "changed") {
      setBackfilling({ ...open, impact: await previewOf(changeFor(open.draft)), changed: true });
      return;
    }
    setBackfilling(undefined);
    await change.done();
  };

  return {
    save,
    dialogs: (
      <>
        <BackfillDialog
          shown={shown}
          backfilling={backfilling}
          onSave={commit}
          onClose={() => {
            setBackfilling(undefined);
          }}
        />
        {sheet.sheet}
      </>
    ),
  };
}
