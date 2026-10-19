/**
 * Why the folder can't be a root
 */
export type RootsRefusalReason =
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
  | "node_offline";
