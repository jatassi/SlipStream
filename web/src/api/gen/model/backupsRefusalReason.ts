/**
 * Why the folder can't hold Backups
 */
export type BackupsRefusalReason =
  | "not_absolute"
  | "not_found"
  | "not_a_folder"
  | "unwritable"
  | "inside_root"
  | "contains_root"
  | "inside_data"
  | "contains_data"
  | "inside_downloads"
  | "contains_downloads"
  | "unresponsive"
  | "network_share"
  | "outside"
  | "node_offline";
