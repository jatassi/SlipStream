/**
 * none: not checked yet; asked: waiting for its Node; no_local_file: this machine has no file of the Backup to check it against
 */
export type BackupsCheckResult =
  | "none"
  | "asked"
  | "ok"
  | "mismatch"
  | "unreadable"
  | "no_local_file";
