import { useState } from "react";

import { sentenceOf } from "../api/errors";
import { cancelGrab } from "../api/gen/query/grabs/grabs";
import { Checkbox } from "../ui/checkbox";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Data } from "../ui/data";
import { toast } from "../ui/toast";
import { Stops } from "./cancel-preview";
import type { Job } from "./job";

export function CancelGrab({
  open,
  onOpenChange,
  ...job
}: Job & {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { grab, work } = job;
  const [blocklist, setBlocklist] = useState(false);
  return (
    <ConfirmDialog
      title="Cancel this Grab?"
      confirmLabel="Cancel Grab"
      dismissLabel="Keep Grab"
      look="fail"
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={async () => {
        try {
          await cancelGrab(grab.id, { blocklist });
        } catch (error) {
          throw new Error(sentenceOf(error), { cause: error });
        }
        toast(`Canceled the Grab of ${work}.`);
      }}
    >
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="text-ink-1">{work}</span>
        <Data className="wrap-anywhere text-meta text-ink-3">{grab.releaseTitle}</Data>
      </div>
      <Stops {...job} />
      {grab.actions.blocklistable && (
        <Checkbox
          label="Also blocklist this Release"
          checked={blocklist}
          onCheckedChange={setBlocklist}
        />
      )}
    </ConfirmDialog>
  );
}
