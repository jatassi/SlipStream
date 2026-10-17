import { useState } from "react";

import { DialogActions } from "../account/dialog-actions";
import type { WorksCandidate, WorksContentType } from "../api/gen/model";
import { useListProfilesSuspense } from "../api/gen/query/targets/targets";
import { Button } from "../ui/button";
import { Dialog, DialogClose } from "../ui/dialog";
import { Switch } from "../ui/switch";
import { useAdding } from "./adding";
import { ProfilePicker } from "./profile-picker";

function Monitor({
  type,
  checked,
  onCheckedChange,
}: {
  type: WorksContentType;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <div className="flex flex-col gap-1">
      <Switch label="Monitor" checked={checked} onCheckedChange={onCheckedChange} />
      <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        A monitored {type.work.one} is searched for whatever its Profile wants.
      </p>
    </div>
  );
}

function AddForm({ type, candidate }: { type: WorksContentType; candidate: WorksCandidate }) {
  const profiles = useListProfilesSuspense().data.profiles.filter(
    (profile) => profile.contentType === type.id,
  );
  const [profileId, setProfileId] = useState(
    (profiles.find((profile) => profile.default) ?? profiles[0])?.id,
  );
  const [monitor, setMonitor] = useState(true);
  const addWork = useAdding();
  return (
    <div className="flex flex-col gap-4">
      {profileId !== undefined && (
        <ProfilePicker profiles={profiles} value={profileId} onValueChange={setProfileId} />
      )}
      <Monitor type={type} checked={monitor} onCheckedChange={setMonitor} />
      <DialogActions>
        <DialogClose>
          <Button>Cancel</Button>
        </DialogClose>
        <Button
          look="primary"
          busy={addWork.isPending}
          onClick={() => {
            addWork.mutate({
              data: {
                contentType: type.id,
                key: candidate.key,
                monitor,
                ...(profileId !== undefined && { profileId }),
              },
            });
          }}
        >
          Add {type.work.one}
        </Button>
      </DialogActions>
    </div>
  );
}

/** Add opens a sheet to pick the new Work's Profile, its type's default first, and whether to monitor it. */
export function AddSheet({
  type,
  candidate,
}: {
  type: WorksContentType;
  candidate: WorksCandidate;
}) {
  return (
    <Dialog
      title={`Add ${candidate.label}`}
      trigger={
        <Button aria-label={`Add ${candidate.label}`} size="small">
          Add
        </Button>
      }
    >
      <AddForm type={type} candidate={candidate} />
    </Dialog>
  );
}
