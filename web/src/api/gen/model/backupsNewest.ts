import type { BackupsCheck } from "./backupsCheck.ts";

export interface BackupsNewest {
  /** @pattern ^[0-9a-f]{32}$ */
  backupId: string;
  /** Its last check */
  check: BackupsCheck;
  copiedAt: string;
  /** The Backup's file name */
  name: string;
}
