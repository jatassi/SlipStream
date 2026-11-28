/**
 * root is a library root, backup a Backup destination, backups_folder the Controller's Backups folder, download_folder a Download Client's folder, and scratch a Node's scratch folder for Conversions
 */
export type TransfersDestinationKind =
  | "root"
  | "backup"
  | "backups_folder"
  | "download_folder"
  | "scratch";
