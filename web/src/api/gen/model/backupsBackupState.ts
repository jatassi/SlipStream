/**
 * verified: it passed its check; failed: it counts for nothing and goes at the next Backup
 */
export type BackupsBackupState = "taking" | "verified" | "failed" | "deleting";
