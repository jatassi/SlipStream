import type { BackupsBackupCheckResult } from "./backupsBackupCheckResult.ts";

export interface BackupsBackupCheck {
  /**
   * When the result came in; null while none has
   * @nullable
   */
  at: string | null;
  /** none: not checked yet; asked: waiting for its Node; no_local_file: this machine has no file of the Backup to check it against */
  result: BackupsBackupCheckResult;
}
