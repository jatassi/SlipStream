import type { BackupsBackup } from "./backupsBackup.ts";
import type { BackupsFailure } from "./backupsFailure.ts";
import type { BackupsPassphraseState } from "./backupsPassphraseState.ts";
import type { BackupsSheetState } from "./backupsSheetState.ts";

export interface BackupsBackupList {
  /** Newest first */
  backups: BackupsBackup[];
  /** Backups failing since the newest good one; null when the newest passed */
  failure: BackupsFailure | null;
  passphrase: BackupsPassphraseState;
  sheet: BackupsSheetState;
  /** Bytes the Backups in this machine's Backups folder take */
  used: number;
}
