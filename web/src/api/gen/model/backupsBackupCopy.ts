import type { BackupsBackupCheck } from "./backupsBackupCheck.ts";
import type { BackupsBackupCopyState } from "./backupsBackupCopyState.ts";

export interface BackupsBackupCopy {
  /** Its last check */
  check: BackupsBackupCheck;
  /**
   * Null until it is copied
   * @nullable
   */
  copiedAt: string | null;
  /** @pattern ^[0-9a-f]{32}$ */
  destinationId: string;
  state: BackupsBackupCopyState;
}
