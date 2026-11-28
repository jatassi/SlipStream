import { type ReactElement, useState } from "react";

import type { TargetsTarget } from "../api/gen/model";
import { updateTarget } from "../api/gen/query/targets/targets";
import { toast } from "../ui/toast";
import { type TargetDraft, bodyOf } from "./draft";
import { useImpactSheet } from "./impact-sheet";
import { type Remaking, RemakeDialog } from "./remake-dialog";
import type { ImpactChange } from "./use-impact-save";

const changeOf = (
  target: TargetsTarget,
  draft: TargetDraft,
  works: readonly string[],
  count: number,
  onSaved: () => Promise<void> | void,
): ImpactChange => ({
  title: `Save ${target.name}`,
  action: "Save",
  run: (changing) =>
    updateTarget(
      target.id,
      { ...bodyOf(draft), ...(works.length > 0 && { remake: [...works] }) },
      { ...changing, ...(count > 0 && { confirmRemake: count }) },
    ),
  done: async () => {
    toast(`Saved ${draft.name.trim()}.`);
    await onSaved();
  },
});

/**
 * Saves a Target through the remake dialog while files made with older
 * Conversion settings remain, then through the Impact sheet when the
 * change reaches Works. `remake` opens the dialog on its own.
 */
export function useTargetSave(
  target: TargetsTarget,
  onSaved: () => Promise<void> | void,
): {
  readonly save: (draft: TargetDraft) => Promise<void>;
  readonly remake: (draft: TargetDraft) => Promise<void>;
  readonly dialogs: ReactElement;
} {
  const impact = useImpactSheet();
  const [remaking, setRemaking] = useState<Remaking>();

  const commit = (draft: TargetDraft, works: readonly string[], count: number) =>
    impact.save(changeOf(target, draft, works, count, onSaved));

  const offer = async (draft: TargetDraft, fromSave: boolean) => {
    const { remake } = (await updateTarget(target.id, bodyOf(draft), { preview: true })).impact;
    if (fromSave && remake.older === 0) {
      await commit(draft, [], 0);
      return;
    }
    setRemaking({
      name: target.name,
      fromSave,
      remake,
      save: (works, count) => commit(draft, works, count),
    });
  };

  return {
    save: (draft) => offer(draft, true),
    remake: (draft) => offer(draft, false),
    dialogs: (
      <>
        <RemakeDialog
          remaking={remaking}
          onClose={() => {
            setRemaking(undefined);
          }}
        />
        {impact.sheet}
      </>
    ),
  };
}
