import { useState } from "react";

import { sentenceOf } from "../../api/errors";
import type { BackupsDestination } from "../../api/gen/model";
import { removeBackupDestination } from "../../api/gen/query/backups/backups";
import { Button } from "../../ui/button";
import { ConfirmDialog } from "../../ui/confirm-dialog";
import { Data } from "../../ui/data";
import { toast } from "../../ui/toast";

export function RemoveButton({
  destination,
  node,
}: {
  destination: BackupsDestination;
  node: string;
}) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        disabled={destination.removing}
        onClick={() => {
          setOpen(true);
        }}
      >
        Remove
      </Button>
      <ConfirmDialog
        title={`Remove the Backup destination on ${node}?`}
        confirmLabel="Remove"
        look="fail"
        open={open}
        onOpenChange={setOpen}
        onConfirm={async () => {
          try {
            await removeBackupDestination(destination.id);
          } catch (error) {
            throw new Error(sentenceOf(error), { cause: error });
          }
          toast(`Removed the Backup destination on ${node}.`);
        }}
      >
        <p>
          SlipStream stops copying Backups to <Data>{destination.folder}</Data>. Copies already in
          that folder stay there.
        </p>
      </ConfirmDialog>
    </>
  );
}
