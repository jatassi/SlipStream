import { useQueryClient } from "@tanstack/react-query";

import { sentenceOf } from "../api/errors";
import type { TrackersTracker } from "../api/gen/model";
import { removeTracker } from "../api/gen/query/trackers/trackers";
import { adoptRemoved } from "../live/adopt-trackers";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { toast } from "../ui/toast";

/** Remove a Tracker that nothing uses. A refusal says what still uses it. */
export function RemoveTrackerDialog({
  tracker,
  onClose,
  onRemoved,
}: {
  tracker: TrackersTracker;
  onClose: () => void;
  /** Runs before the Tracker's page data leaves the cache, such as to leave its page. */
  onRemoved?: (() => Promise<unknown>) | undefined;
}) {
  const queryClient = useQueryClient();
  return (
    <ConfirmDialog
      title={`Remove ${tracker.name}`}
      confirmLabel="Remove"
      look="fail"
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      onConfirm={async () => {
        try {
          await removeTracker(tracker.id);
        } catch (error) {
          throw new Error(sentenceOf(error), { cause: error });
        }
        const forget = adoptRemoved(queryClient, tracker.id);
        await onRemoved?.();
        forget();
        toast(`Removed ${tracker.name}`);
      }}
    >
      <p>
        Removing {tracker.name} also deletes its seeding rule, its goal and its seeding history. Its
        finished Grabs and removed torrents stay, with no Tracker.
      </p>
    </ConfirmDialog>
  );
}
