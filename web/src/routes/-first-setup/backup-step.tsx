import { Suspense } from "react";

import {
  prepareRecoverySheet,
  saveBackupPassphrase,
  useGetBackupPassphraseSuspense,
} from "../../api/gen/query/backups/backups";
import { recoverySheetUrl } from "../../api/urls";
import { useAttempt } from "../../targets/attempt";
import { Button } from "../../ui/button";
import { Data } from "../../ui/data";
import { toast } from "../../ui/toast";
import { StepFooter } from "./frame";
import type { StepBodyProps } from "./screens";

// The PIN prompt comes from the pre-flight, since a link the browser follows
// can't show it.
async function downloadSheet() {
  await prepareRecoverySheet();
  const link = document.createElement("a");
  link.href = recoverySheetUrl;
  link.download = "";
  link.click();
}

function Passphrase({ nav }: StepBodyProps) {
  const { data } = useGetBackupPassphraseSuspense();
  const download = useAttempt(downloadSheet);
  const saved = useAttempt(async () => {
    await saveBackupPassphrase();
    await nav.complete();
  });
  const copy = useAttempt(async () => {
    await navigator.clipboard.writeText(data.passphrase);
    toast("Copied the passphrase.");
  });
  return (
    <>
      <p>
        SlipStream keeps a copy and uses it for every scheduled Backup. You need this passphrase
        only to restore a Backup on another machine. Store it in a password manager or print it.
      </p>
      <p>
        <Data className="break-words">{data.passphrase}</Data>
      </p>
      {copy.failure}
      {download.failure}
      {saved.failure}
      <div className="flex flex-wrap gap-2">
        <Button busy={copy.busy} onClick={copy.run}>
          Copy
        </Button>
        <Button busy={download.busy} onClick={download.run}>
          Download recovery sheet
        </Button>
      </div>
      <StepFooter
        nav={nav}
        primary={
          <Button look="primary" busy={saved.busy} onClick={saved.run}>
            I saved it
          </Button>
        }
      />
    </>
  );
}

export function BackupStep({ step, nav }: StepBodyProps) {
  if (step.state === "done") {
    return (
      <>
        <p>You saved the Backup passphrase.</p>
        <StepFooter nav={nav} />
      </>
    );
  }
  return (
    <Suspense>
      <Passphrase step={step} nav={nav} />
    </Suspense>
  );
}
