import { type QueryClient, useQueryClient } from "@tanstack/react-query";

import { sentenceOf } from "../api/errors";
import { clearHistory } from "../api/gen/query/history/history";
import { adoptHistoryCleared } from "../live/history-effects";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { toast } from "../ui/toast";

const words =
  "Every History entry is deleted, and the bell's list is emptied too. A message that is still being sent stays until it's sent. Grabs, Imports and files aren't deleted. This can't be undone.";

async function clear(queryClient: QueryClient) {
  try {
    await clearHistory();
  } catch (error) {
    toast(`History wasn't cleared. ${sentenceOf(error)}`, { kind: "fail" });
    return;
  }
  toast("Clearing History.");
  adoptHistoryCleared(queryClient);
}

/** Clear History, once confirmed. History lists nothing at once and the sweep finishes behind it. */
export function ClearDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const queryClient = useQueryClient();
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
      onConfirm={() => clear(queryClient)}
    >
      <p>{words}</p>
    </ConfirmDialog>
  );
}
