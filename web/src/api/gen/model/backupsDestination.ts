import type { BackupsCopies } from "./backupsCopies.ts";
import type { BackupsNewest } from "./backupsNewest.ts";

export interface BackupsDestination {
  addedAt: string;
  /** Its copies of the listed Backups, by state */
  copies: BackupsCopies;
  /** As the Admin typed it */
  folder: string;
  /** @pattern ^[0-9a-f]{32}$ */
  id: string;
  /** Its newest copy, null before one is copied */
  newest: BackupsNewest | null;
  nodeId: string;
  /** Removed, while a copy already running into its folder finishes */
  removing: boolean;
}
