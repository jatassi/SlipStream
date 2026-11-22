import { useQueryClient } from "@tanstack/react-query";

import { sentenceOf } from "../../api/errors";
import type { BackupsBackup } from "../../api/gen/model";
import { deleteBackup } from "../../api/gen/query/backups/backups";
import type { Clock } from "../../ui/clock";
import { adoptBackupList } from "../../live/adopt-backups";
import { ConfirmDialog } from "../../ui/confirm-dialog";
import { toast } from "../../ui/toast";
import { deletedToast } from "./words";

export function DeleteBackupDialog({
  backup,
  from,
  clock,
  open,
  onOpenChange,
}: {
  backup: BackupsBackup;
  from: string;
  clock: Clock;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  return (
    <ConfirmDialog
      title={`Delete the Backup ${from}?`}
      confirmLabel="Delete"
      look="fail"
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={async () => {
        try {
          adoptBackupList(queryClient, await deleteBackup(backup.id));
        } catch (error) {
          throw new Error(sentenceOf(error), { cause: error });
        }
        toast(deletedToast(backup, clock));
      }}
    >
      <p>SlipStream deletes it here and its copy on every Backup destination.</p>
    </ConfirmDialog>
  );
}
