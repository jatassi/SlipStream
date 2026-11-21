import type { BackupsCheckResult } from "./backupsCheckResult.ts";

export interface BackupsCheck {
  /**
   * When the result came in; null while none has
   * @nullable
   */
  at: string | null;
  /** Why the check failed, as a sentence */
  cause?: string;
  /** none: not checked yet; asked: waiting for its Node; no_local_file: this machine has no file of the Backup to check it against */
  result: BackupsCheckResult;
}
