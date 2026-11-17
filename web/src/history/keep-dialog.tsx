import { useSuspenseQuery } from "@tanstack/react-query";

import { getGetTaskSuspenseQueryOptions, useUpdateTask } from "../api/gen/query/tasks/tasks";
import { Dialog } from "../ui/dialog";
import { SettingField } from "../ui/setting-field";

/** Keep history for, which Housekeeping holds as its Tuning. */
const keepKey = "history.keepFor";

function KeepField() {
  const { tuning } = useSuspenseQuery(getGetTaskSuspenseQueryOptions("housekeeping")).data;
  const update = useUpdateTask();
  const keep = tuning.find((field) => field.key === keepKey);
  if (keep === undefined) {
    throw new Error("Housekeeping has no Keep history for");
  }
  return (
    <SettingField
      field={keep}
      onSave={(value) =>
        update.mutateAsync({ taskId: "housekeeping", data: { tuning: { [keepKey]: value } } })
      }
    />
  );
}

/** Keep history for, the one stored value Housekeeping prunes History by, under a title that isn't its label again. */
export function KeepDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <Dialog
      title="History"
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onClose();
        }
      }}
    >
      {open && <KeepField />}
    </Dialog>
  );
}
