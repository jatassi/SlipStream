import { Suspense } from "react";

import {
  saveBackupPassphrase,
  useGetBackupPassphraseSuspense,
} from "../../api/gen/query/backups/backups";
import { downloadSheet } from "../../settings/backups/downloads";
import { ShownPassphrase } from "../../settings/backups/passphrase";
import { useAttempt } from "../../targets/attempt";
import { Button } from "../../ui/button";
import { StepFooter } from "./frame";
import type { StepBodyProps } from "./screens";

function Passphrase({ nav }: StepBodyProps) {
  const { data } = useGetBackupPassphraseSuspense();
  const download = useAttempt(downloadSheet);
  const saved = useAttempt(async () => {
    await saveBackupPassphrase();
    await nav.complete();
  });
  return (
    <>
      <ShownPassphrase passphrase={data.passphrase} />
      {download.failure}
      {saved.failure}
      <div>
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
