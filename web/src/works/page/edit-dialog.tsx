import { useState } from "react";

import type { WorksWorkView } from "../../api/gen/model";
import { useListProfilesSuspense } from "../../api/gen/query/targets/targets";
import { AttemptActions, useAttempt } from "../../targets/attempt";
import type { ImpactChange } from "../../targets/use-impact-save";
import { Dialog } from "../../ui/dialog";
import { act } from "../actions/act";
import type { Subject } from "../actions/actions";
import { profileChange } from "../actions/profile-change";
import { ProfilePicker } from "../profile-picker";
// oxlint-disable-next-line import/max-dependencies -- Edit changes the Profile through the Impact, the root, and the type's own options
import { LevelFolders, OptionsFields, RenameLine, saveLevelFolders } from "./options-fields";
import { RootField } from "./root-field";

type EditDialogProps = {
  readonly subject: Subject;
  readonly work: WorksWorkView;
  /** Its library files, Kept files included. */
  readonly files: number;
  readonly save: (change: ImpactChange) => Promise<void>;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  /** Opens the Work's rename plan in the dialog's place. */
  readonly onRename: () => void;
};

/**
 * The dialog's fields, saved together: the root, the level folders, then a
 * new Profile through the Impact. A level folders change that leaves files
 * in their old folders calls onMoved and keeps the dialog open.
 */
function useFields({
  subject,
  work,
  files,
  save,
  onOpenChange,
  onMoved,
}: Omit<EditDialogProps, "open" | "onRename"> & { onMoved: () => void }) {
  const profiles = useListProfilesSuspense().data.profiles.filter(
    (profile) => profile.contentType === subject.type.id,
  );
  const profileId = work.profile.id;
  const rootId = work.root?.id;
  const [profile, setProfile] = useState(profileId);
  const [root, setRoot] = useState(rootId ?? "");
  const [levelFolders, setLevelFolders] = useState(work.levelFolders);
  const close = () => {
    onOpenChange(false);
  };
  const saving = useAttempt(async () => {
    if (root !== "" && root !== rootId) {
      await act([subject], { kind: "root", rootId: root });
    }
    const moved = levelFolders !== work.levelFolders;
    if (moved) {
      await saveLevelFolders(work.id, levelFolders);
    }
    if (profile !== profileId) {
      const name = profiles.find(({ id }) => id === profile)?.name ?? "";
      const title = `Give ${subject.label} the Profile ${name}`;
      await save(profileChange({ subjects: [subject], profileId: profile, title, done: close }));
    } else if (moved && files > 0) {
      onMoved();
    } else {
      close();
    }
  });
  return {
    profiles,
    profile: [profile, setProfile] as const,
    root: [root, setRoot] as const,
    levelFolders: [levelFolders, setLevelFolders] as const,
    close,
    saving,
  };
}

/**
 * The Work's Profile, saved through the Impact, its library root while it
 * has no files, and for a Work of many Items its level folders and its
 * type's options. Once a change leaves files with old names or folders, it
 * links to the Work's rename plan.
 */
export function EditDialog({ open, onRename, ...props }: EditDialogProps) {
  const { subject, work, files, onOpenChange } = props;
  const [renames, setRenames] = useState(false);
  const renamed = () => {
    setRenames(files > 0);
  };
  const fields = useFields({ ...props, onMoved: renamed });
  const [profile, setProfile] = fields.profile;
  const [root, setRoot] = fields.root;
  const [levelFolders, setLevelFolders] = fields.levelFolders;
  return (
    <Dialog title={`Edit ${subject.label}`} open={open} onOpenChange={onOpenChange}>
      <div className="flex flex-col gap-4">
        <ProfilePicker profiles={fields.profiles} value={profile} onValueChange={setProfile} />
        <RootField type={subject.type} value={root} files={files} onValueChange={setRoot} />
        <LevelFolders type={subject.type} on={levelFolders} onChange={setLevelFolders} />
        <OptionsFields type={subject.type} work={work} onSaved={renamed} />
        {renames && <RenameLine onRename={onRename} />}
        <p className="text-ink-2">Before a new Profile is saved, the Impact lists what changes.</p>
        <AttemptActions attempt={fields.saving} onCancel={fields.close}>
          Save
        </AttemptActions>
      </div>
    </Dialog>
  );
}
