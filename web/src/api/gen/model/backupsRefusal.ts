import type { BackupsRefusalReason } from "./backupsRefusalReason.ts";

export interface BackupsRefusal {
  /** Why the folder can't hold Backups */
  reason: BackupsRefusalReason;
}
