/**
 * root: a library root's rules, on a Storage Node; downloads: a downloads folder's, writable, on no network share and apart from every root on the Node; import: a folder to import from, inside one of the Node's library roots or download folders; backups: a folder for Backups, writable and apart from every root and downloads folder on the Node; converting: a Node's scratch folder for Conversions, writable and apart from every root, downloads folder and the Backups folder on the Node
 */
export type CheckFolderInputBodyPurpose =
  | "root"
  | "downloads"
  | "import"
  | "backups"
  | "converting";
