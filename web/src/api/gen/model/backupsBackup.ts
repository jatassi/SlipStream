import type { BackupsBackupCopy } from "./backupsBackupCopy.ts";
import type { BackupsBackupState } from "./backupsBackupState.ts";
import type { BackupsBackupTrigger } from "./backupsBackupTrigger.ts";

export interface BackupsBackup {
  /** Its copy on each destination, in the order the destinations were added */
  copies: BackupsBackupCopy[];
  /** @pattern ^[0-9a-f]{32}$ */
  id: string;
  /** Its file is in this machine's Backups folder */
  local: boolean;
  /** The Backup's file name */
  name: string;
  /** Bytes */
  size: number;
  /** verified: it passed its check; failed: it didn't pass its check, and the next Backup deletes it */
  state: BackupsBackupState;
  takenAt: string;
  /** Why it was taken */
  trigger: BackupsBackupTrigger;
  /** The SlipStream version that took it */
  version: string;
  /** Taken while SlipStream was Reconciling after a restore */
  whileReconciling: boolean;
}
