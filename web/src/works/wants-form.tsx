import { useState } from "react";

import type { TargetsProfile, TargetsWants, WorksTree } from "../api/gen/model";
import { setWorkWants } from "../api/gen/query/targets/targets";
import { AttemptActions, useAttempt } from "../targets/attempt";
import { savingCuts } from "../targets/editions-draft";
import type { ImpactChange } from "../targets/use-impact-save";
import { toast } from "../ui/toast";
import { ProfilePicker } from "./profile-picker";
import { wantsBodyOf, wantsDraftOf } from "./wants-draft";
import { WantsEditions } from "./wants-editions";

export function WantsForm({
  work,
  profiles,
  wants,
  save,
  onDone,
}: {
  work: WorksTree;
  profiles: readonly TargetsProfile[];
  wants: TargetsWants;
  save: (change: ImpactChange) => Promise<void>;
  onDone: () => void;
}) {
  const [draft, setDraft] = useState(() => wantsDraftOf(wants));
  const profile = profiles.find((candidate) => candidate.id === draft.profileId);
  const saving = useAttempt(async () => {
    const body = wantsBodyOf(draft);
    await save({
      title: `Save ${work.label}`,
      action: "Save",
      run: (changing) => savingCuts(() => setWorkWants(work.id, body, changing)),
      done: () => {
        toast(`Saved what ${work.label} wants.`);
        onDone();
      },
    });
  });
  return (
    <div className="flex flex-col gap-5">
      <ProfilePicker
        profiles={profiles}
        value={draft.profileId}
        onValueChange={(profileId) => {
          setDraft({ ...draft, profileId });
        }}
      />
      {profile !== undefined && (
        <WantsEditions draft={draft} profile={profile} onChange={setDraft} />
      )}
      <AttemptActions attempt={saving} onCancel={onDone}>
        Save
      </AttemptActions>
    </div>
  );
}
