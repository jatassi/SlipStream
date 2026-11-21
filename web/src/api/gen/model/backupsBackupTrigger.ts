/**
 * Why it was taken
 */
export type BackupsBackupTrigger =
  | "scheduled"
  | "manual"
  | "update"
  | "before-restore"
  | "before-import"
  | "before-handover"
  | "superseded"
  | "restored";
