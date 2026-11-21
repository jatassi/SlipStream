import { useState } from "react";

import { sentenceOf } from "../../api/errors";
import { retireController } from "../../api/gen/query/backups/backups";
import { Button } from "../../ui/button";
import { ConfirmDialog } from "../../ui/confirm-dialog";

export function RetireButton() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        look="fail"
        onClick={() => {
          setOpen(true);
        }}
      >
        Retire this Controller…
      </Button>
      <ConfirmDialog
        title="Retire this Controller?"
        confirmLabel="Retire"
        look="fail"
        open={open}
        onOpenChange={setOpen}
        onConfirm={async () => {
          try {
            await retireController();
          } catch (error) {
            throw new Error(sentenceOf(error), { cause: error });
          }
        }}
      >
        <p>
          Do this only on an old Controller whose Backup you restored on another machine. Its
          torrents pause and its searches stop, so it can&apos;t seed the same torrents as the new
          Controller.
        </p>
      </ConfirmDialog>
    </>
  );
}
