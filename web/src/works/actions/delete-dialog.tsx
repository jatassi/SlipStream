import { useState } from "react";

import type { WorksDeletingFiles } from "../../api/gen/model";
import { ConfirmDialog } from "../../ui/confirm-dialog";
import { RadioGroup } from "../../ui/radio";
import { act } from "./act";
import type { Subject } from "./actions";
import { DeletePreview, useDeletePreview } from "./delete-preview";
import { namedOf } from "./outcomes";

const choices = [
  { value: "delete", label: "Delete files from disk" },
  { value: "leave", label: "Leave files on disk" },
] as const;

type DeleteDialogProps = {
  readonly subjects: readonly Subject[];
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  /** Runs as the Admin confirms, before the request, such as for a page that then leaves without a second toast. */
  readonly onDeleting?: () => void;
};

/**
 * Asks before deleting Works, with what the delete takes: their files and
 * Kept files with their size, and what keeps running in the torrent
 * client. Deleting files is the default. A Work whose files sit on an
 * offline Node can't lose them, so the dialog names it and offers leaving
 * the files instead.
 */
export function DeleteDialog({ subjects, open, onOpenChange, onDeleting }: DeleteDialogProps) {
  const [files, setFiles] = useState<WorksDeletingFiles>("delete");
  const preview = useDeletePreview(subjects, files);
  const named = namedOf(subjects);
  return (
    <ConfirmDialog
      title={`Delete ${named.words}?`}
      confirmLabel="Delete"
      look="fail"
      confirmDisabled={(preview.data?.plan.works ?? 0) === 0}
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={() => {
        onDeleting?.();
        return act(subjects, { kind: "delete", files });
      }}
    >
      <RadioGroup label="Files" options={choices} value={files} onValueChange={setFiles} />
      <DeletePreview
        subjects={subjects}
        named={named}
        files={files}
        preview={preview}
        onLeave={() => {
          setFiles("leave");
        }}
      />
    </ConfirmDialog>
  );
}
