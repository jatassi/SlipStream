import type { BackupsBackupList } from "../../api/gen/model";
import { backUpNow } from "../../api/gen/query/backups/backups";
import { useAttempt } from "../../targets/attempt";
import { Button } from "../../ui/button";
import { Section } from "../../ui/section";
import { toast } from "../../ui/toast";
import { BackupRow, type Rows } from "./backup-row";

function BackUpNow() {
  const start = useAttempt(async () => {
    await backUpNow();
    toast("Backing up…");
  });
  return (
    <div className="flex flex-col items-end gap-1">
      <Button look="primary" busy={start.busy} onClick={start.run}>
        Back up now
      </Button>
      {start.failure}
    </div>
  );
}

export function BackupsList({ list, ...rows }: Rows & { list: BackupsBackupList }) {
  return (
    <Section title="Backups" count={list.backups.length} action={<BackUpNow />}>
      {list.backups.length === 0 ? (
        <p className="text-ink-2">No Backups yet.</p>
      ) : (
        <ul aria-label="Backups" className="rounded-card bg-glass ring-1 ring-hairline ring-inset">
          {list.backups.map((backup) => (
            <BackupRow key={backup.id} backup={backup} {...rows} />
          ))}
        </ul>
      )}
    </Section>
  );
}
