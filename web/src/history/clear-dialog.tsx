import { sentenceOf } from "../api/errors";
import { clearHistory } from "../api/gen/query/history/history";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { toast } from "../ui/toast";

const words =
  "Every History entry is deleted, and the bell's list is emptied too. A message that is still being sent stays until it's sent. Grabs, Imports and files aren't deleted. This can't be undone.";

async function clear() {
  try {
    await clearHistory();
    toast("Clearing History.");
  } catch (error) {
    toast(`History wasn't cleared. ${sentenceOf(error)}`, { kind: "fail" });
  }
}

/** Clear History, once confirmed. History lists nothing at once and the sweep finishes behind it. */
export function ClearDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <ConfirmDialog
      title="Clear History?"
      confirmLabel="Clear History"
      look="fail"
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onClose();
        }
      }}
      onConfirm={clear}
    >
      <p>{words}</p>
    </ConfirmDialog>
  );
}
