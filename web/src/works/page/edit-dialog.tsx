import { useState } from "react";

import { useListProfilesSuspense } from "../../api/gen/query/targets/targets";
import { AttemptActions, useAttempt } from "../../targets/attempt";
import type { ImpactChange } from "../../targets/use-impact-save";
import { Dialog } from "../../ui/dialog";
import { act } from "../actions/act";
import type { Subject } from "../actions/actions";
import { profileChange } from "../actions/profile-change";
import { ProfilePicker } from "../profile-picker";
import { RootField } from "./root-field";

type EditDialogProps = {
  readonly subject: Subject;
  readonly profileId: string;
  readonly rootId: string | undefined;
  /** Its library files, Kept files included. */
  readonly files: number;
  readonly save: (change: ImpactChange) => Promise<void>;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
};

/**
 * The Work's Profile, saved through the Impact, and its library root while
 * it has no files.
 */
export function EditDialog({
  subject,
  profileId,
  rootId,
  files,
  save,
  open,
  onOpenChange,
}: EditDialogProps) {
  const profiles = useListProfilesSuspense().data.profiles.filter(
    (profile) => profile.contentType === subject.type.id,
  );
  const [profile, setProfile] = useState(profileId);
  const [root, setRoot] = useState(rootId ?? "");
  const close = () => {
    onOpenChange(false);
  };
  const saving = useAttempt(async () => {
    if (root !== "" && root !== rootId) {
      await act([subject], { kind: "root", rootId: root });
    }
    if (profile === profileId) {
      close();
      return;
    }
    const name = profiles.find(({ id }) => id === profile)?.name ?? "";
    await save(
      profileChange({
        subjects: [subject],
        profileId: profile,
        title: `Give ${subject.label} the Profile ${name}`,
        done: close,
      }),
    );
  });
  return (
    <Dialog title={`Edit ${subject.label}`} open={open} onOpenChange={onOpenChange}>
      <div className="flex flex-col gap-4">
        <ProfilePicker profiles={profiles} value={profile} onValueChange={setProfile} />
        <RootField type={subject.type} value={root} files={files} onValueChange={setRoot} />
        <p className="text-ink-2">A new Profile shows what changes before anything is saved.</p>
        <AttemptActions attempt={saving} onCancel={close}>
          Save
        </AttemptActions>
      </div>
    </Dialog>
  );
}
