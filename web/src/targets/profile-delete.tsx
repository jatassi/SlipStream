import { useState } from "react";

import type { TargetsProfile } from "../api/gen/model";
import { deleteProfile, useListProfilesSuspense } from "../api/gen/query/targets/targets";
import { Dialog } from "../ui/dialog";
import { Select } from "../ui/select";
import { toast } from "../ui/toast";
import { AttemptActions, useAttempt } from "./attempt";
import { useImpactSheet } from "./impact-sheet";
import type { ImpactChange } from "./use-impact-save";

type Replacements = readonly [TargetsProfile, ...TargetsProfile[]];

const movingOf = (count: number) =>
  `${count === 1 ? "Its 1 Work moves" : `Its ${String(count)} Works move`} to the Profile you pick.`;

function DeleteForm({
  profile,
  replacements,
  save,
  onDone,
}: {
  profile: TargetsProfile;
  replacements: Replacements;
  save: (change: ImpactChange) => Promise<void>;
  onDone: () => void;
}) {
  const [moveTo, setMoveTo] = useState(
    (replacements.find((candidate) => candidate.default) ?? replacements[0]).id,
  );
  const into = replacements.find((candidate) => candidate.id === moveTo)?.name ?? "";
  const deleting = useAttempt(() =>
    save({
      title: `Delete ${profile.name}`,
      action: `Delete ${profile.name}`,
      run: (changing) => deleteProfile(profile.id, { moveTo, ...changing }),
      done: () => {
        toast(`Deleted ${profile.name}. Its Works now take ${into}.`);
        onDone();
      },
    }),
  );
  return (
    <div className="flex flex-col gap-4">
      <p className="text-ink-2">{movingOf(profile.works)}</p>
      <Select
        label="Move its Works to"
        options={replacements.map((candidate) => ({ value: candidate.id, label: candidate.name }))}
        value={moveTo}
        onValueChange={setMoveTo}
      />
      <AttemptActions attempt={deleting} look="fail" onCancel={onDone}>
        Delete {profile.name}
      </AttemptActions>
    </div>
  );
}

export function DeleteProfileDialog({
  profile,
  open,
  onOpenChange,
}: {
  profile: TargetsProfile;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [first, ...rest] = useListProfilesSuspense().data.profiles.filter(
    (candidate) => candidate.contentType === profile.contentType && candidate.id !== profile.id,
  );
  const impact = useImpactSheet();
  return (
    <>
      <Dialog title={`Delete ${profile.name}?`} open={open} onOpenChange={onOpenChange}>
        {first !== undefined && (
          <DeleteForm
            profile={profile}
            replacements={[first, ...rest]}
            save={impact.save}
            onDone={() => {
              onOpenChange(false);
            }}
          />
        )}
      </Dialog>
      {impact.sheet}
    </>
  );
}
