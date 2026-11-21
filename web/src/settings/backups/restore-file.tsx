import { useState } from "react";

import {
  confirmBackupRestore,
  openBackupRestore,
  uploadBackupRestore,
} from "../../api/gen/query/restore/restore";
import { type RestoreCalls, useRestoreFlow } from "../../restore/flow";
import { Button } from "../../ui/button";
import { Dialog } from "../../ui/dialog";
import { FlowStep } from "./flow-step";

export const settingsCalls: RestoreCalls = {
  upload: (file, onUploadProgress) => uploadBackupRestore(file, { onUploadProgress }),
  open: (id, passphrase) => openBackupRestore(id, { passphrase }),
  confirm: (id, body) => confirmBackupRestore(id, body),
};

export function RestoreFileButton() {
  const [open, setOpen] = useState(false);
  const flow = useRestoreFlow(settingsCalls);
  const locked = flow.step.at === "uploading" || flow.step.at === "restarting";
  return (
    <>
      <Button
        onClick={() => {
          flow.reset();
          setOpen(true);
        }}
      >
        Restore a file…
      </Button>
      <Dialog
        title="Restore a file"
        open={open}
        onOpenChange={(next) => {
          if (!locked) {
            setOpen(next);
          }
        }}
      >
        <FlowStep
          flow={flow}
          onCancel={() => {
            setOpen(false);
          }}
        />
      </Dialog>
    </>
  );
}
