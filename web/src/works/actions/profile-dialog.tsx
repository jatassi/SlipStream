import { useState } from "react";

import type { WorksContentType } from "../../api/gen/model";
import { useListProfilesSuspense } from "../../api/gen/query/targets/targets";
import { useImpactSheet } from "../../targets/impact-sheet";
import { AttemptActions, useAttempt } from "../../targets/attempt";
import { Dialog } from "../../ui/dialog";
import { ProfilePicker } from "../profile-picker";
import type { Subject } from "./actions";
import { namedOf } from "./outcomes";
import { profileChange } from "./profile-change";

type ProfileDialogProps = {
  readonly type: WorksContentType;
  readonly subjects: readonly Subject[];
  /** The Profile the picker starts on, such as the one Work's own. */
  readonly profileId: string | undefined;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
};

/**
 * Picks a Profile for one Work or many, then shows one Impact for all of
 * them before it saves.
 */
export function ProfileDialog({
  type,
  subjects,
  profileId,
  open,
  onOpenChange,
}: ProfileDialogProps) {
  const profiles = useListProfilesSuspense().data.profiles.filter(
    (profile) => profile.contentType === type.id,
  );
  const fallback = profiles.find((profile) => profile.default) ?? profiles[0];
  const [picked, setPicked] = useState(profileId ?? fallback?.id ?? "");
  const impact = useImpactSheet();
  const named = namedOf(subjects).words;
  const name = profiles.find((profile) => profile.id === picked)?.name ?? "";
  const saving = useAttempt(() =>
    impact.save(
      profileChange({
        subjects,
        profileId: picked,
        title: `Give ${named} the Profile ${name}`,
        done: () => {
          onOpenChange(false);
        },
      }),
    ),
  );
  return (
    <>
      <Dialog title={`Change the Profile of ${named}`} open={open} onOpenChange={onOpenChange}>
        <div className="flex flex-col gap-4">
          <ProfilePicker profiles={profiles} value={picked} onValueChange={setPicked} />
          <p className="text-ink-2">
            A Work with its own Edition list keeps that list. Next shows what changes before
            anything is saved.
          </p>
          <AttemptActions
            attempt={saving}
            onCancel={() => {
              onOpenChange(false);
            }}
          >
            Next
          </AttemptActions>
        </div>
      </Dialog>
      {impact.sheet}
    </>
  );
}
