/**
 * verified: it passed its check; failed: it didn't pass its check, and the next Backup deletes it
 */
export type BackupsBackupState = "taking" | "verified" | "failed" | "deleting";
