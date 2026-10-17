import { isProblem } from "../api/errors";
import type { TargetsCut } from "../api/gen/model";
import { deleteCut } from "../api/gen/query/targets/targets";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { toast } from "../ui/toast";
import type { Opening } from "./cut-dialogs";

/** Deletes the cut, naming what holds it when the Node refuses. */
async function deleting(cut: TargetsCut) {
  try {
    await deleteCut(cut.key);
  } catch (error) {
    if (isProblem(error, "cut_in_use")) {
      throw new Error(`${cut.name} can't be deleted. ${error.detail ?? error.title}.`, {
        cause: error,
      });
    }
    throw error;
  }
  toast(`Deleted ${cut.name}.`);
}

export function DeleteCutDialog({ cut, open, onOpenChange }: Opening & { cut: TargetsCut }) {
  return (
    <ConfirmDialog
      title={`Delete ${cut.name}?`}
      confirmLabel="Delete"
      look="fail"
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={() => deleting(cut)}
    >
      <p>
        SlipStream stops reading it in titles. A cut that a Profile, a Work or a file holds
        can&apos;t be deleted.
      </p>
    </ConfirmDialog>
  );
}
