import { useState } from "react";

import { sentenceOf } from "../../api/errors";
import { replaceBackupPassphrase } from "../../api/gen/query/backups/backups";
import { Button } from "../../ui/button";
import { Checkbox } from "../../ui/checkbox";
import { ConfirmDialog } from "../../ui/confirm-dialog";
import { toast } from "../../ui/toast";

export function ReplaceButton({ replaced }: { replaced: () => void }) {
  const [open, setOpen] = useState(false);
  const [deleteOlder, setDeleteOlder] = useState(true);
  return (
    <>
      <Button
        onClick={() => {
          setDeleteOlder(true);
          setOpen(true);
        }}
      >
        Replace…
      </Button>
      <ConfirmDialog
        title="Replace the passphrase?"
        confirmLabel="Replace"
        open={open}
        onOpenChange={setOpen}
        onConfirm={async () => {
          try {
            await replaceBackupPassphrase({ deleteOlder });
          } catch (error) {
            throw new Error(sentenceOf(error), { cause: error });
          }
          replaced();
          toast("Backing up…");
        }}
      >
        <p>
          SlipStream takes a new Backup with the new passphrase. Backups you already have still need
          the passphrase you have now.
        </p>
        <Checkbox
          label="Delete older Backups and their copies once the new Backup is checked"
          checked={deleteOlder}
          onCheckedChange={setDeleteOlder}
        />
      </ConfirmDialog>
    </>
  );
}
