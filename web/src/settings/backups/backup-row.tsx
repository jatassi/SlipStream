import { useState } from "react";

import type { BackupsBackup, BackupsDestination } from "../../api/gen/model";
import { bytes } from "../../format";
import type { Clock } from "../../ui/clock";
import { RelativeTime } from "../../ui/relative-time";
import { Term } from "../../ui/term";
import { whenWords } from "../../ui/time-words";
import { BackupMenu } from "./backup-menu";
import { DeleteBackupDialog } from "./delete-backup";
import { StateMark, copiesLine } from "./marks";
// oxlint-disable-next-line import/max-dependencies -- a Backup row reads its time on the clock and opens its menu, restore and delete
import { RestoreBackupDialog } from "./restore";
import { type Names, triggerWords } from "./words";

export type Rows = {
  destinations: readonly BackupsDestination[];
  names: Names;
  clock: Clock;
};

function BackupFacts({ backup, copies }: { backup: BackupsBackup; copies: string | undefined }) {
  return (
    <div className="flex min-w-0 flex-1 flex-col gap-0.5">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <span className="font-semibold text-ink-1">
          <RelativeTime at={backup.takenAt} form="clock" />
        </span>
        <StateMark backup={backup} />
      </div>
      <p className="text-meta text-ink-3">
        {triggerWords[backup.trigger]} · SlipStream {backup.version} · {bytes(backup.size)}
        {backup.whileReconciling && (
          <>
            {" "}
            · taken while SlipStream was <Term id="reconciling" />
          </>
        )}
      </p>
      {copies !== undefined && <p className="text-meta text-ink-3">{copies}</p>}
    </div>
  );
}

export function BackupRow({
  backup,
  destinations,
  names,
  clock,
}: Rows & { backup: BackupsBackup }) {
  const [restoring, setRestoring] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const copies = copiesLine(backup, destinations, names);
  const from = whenWords(backup.takenAt, "from", clock);
  return (
    <li className="relative flex items-center gap-3 px-4 py-2.5 not-first:before:absolute not-first:before:inset-x-0 not-first:before:top-0 not-first:before:h-px not-first:before:bg-hairline in-data-[density=touch]:not-first:before:left-4">
      <BackupFacts backup={backup} copies={copies} />
      <BackupMenu
        backup={backup}
        from={from}
        onRestore={() => {
          setRestoring(true);
        }}
        onDelete={() => {
          setDeleting(true);
        }}
      />
      <RestoreBackupDialog
        backup={backup}
        destinations={destinations}
        names={names}
        open={restoring}
        onOpenChange={setRestoring}
      />
      <DeleteBackupDialog
        backup={backup}
        from={from}
        clock={clock}
        open={deleting}
        onOpenChange={setDeleting}
      />
    </li>
  );
}
