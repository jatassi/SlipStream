import type { BackupsBackup, BackupsDestination } from "../../api/gen/model";
import { useRestoreFlow } from "../../restore/flow";
import { InPlaceWords } from "../../restore/summary-step";
import { inPlaceTitle } from "../../restore/words";
import { useClock } from "../../ui/clock";
import { Dialog } from "../../ui/dialog";
import { FlowStep } from "./flow-step";
import { usePrepare } from "./restore-prepare";
import { Preparing } from "./restore-preparing";
import { settingsCalls } from "./restore-file";
// oxlint-disable-next-line import/max-dependencies -- the dialog runs the restore flow and reads its title on the clock
import type { Names } from "./words";

function InPlaceBody({
  backup,
  destinations,
  names,
  onCancel,
}: {
  backup: BackupsBackup;
  destinations: readonly BackupsDestination[];
  names: Names;
  onCancel: () => void;
}) {
  const clock = useClock();
  const flow = useRestoreFlow(settingsCalls);
  const { prepare, pull } = usePrepare(backup, destinations, flow);
  return (
    <div className="flex flex-col gap-4">
      {prepare.at === "staged" ? (
        <FlowStep flow={flow} onCancel={onCancel} />
      ) : (
        <>
          <InPlaceWords takenAt={backup.takenAt} clock={clock} />
          <Preparing prepare={prepare} pull={pull} names={names} onCancel={onCancel} />
        </>
      )}
    </div>
  );
}

export function RestoreBackupDialog({
  backup,
  destinations,
  names,
  open,
  onOpenChange,
}: {
  backup: BackupsBackup;
  destinations: readonly BackupsDestination[];
  names: Names;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const clock = useClock();
  return (
    <Dialog title={inPlaceTitle(backup.takenAt, clock)} open={open} onOpenChange={onOpenChange}>
      {open && (
        <InPlaceBody
          backup={backup}
          destinations={destinations}
          names={names}
          onCancel={() => {
            onOpenChange(false);
          }}
        />
      )}
    </Dialog>
  );
}
