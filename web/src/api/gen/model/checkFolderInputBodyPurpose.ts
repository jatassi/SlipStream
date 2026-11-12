/**
 * root: a library root's rules, on a Storage Node; downloads: a downloads folder's, writable, on no network share and apart from every root on the Node; import: a folder to import from, inside one of the Node's library roots or download folders
 */
export type CheckFolderInputBodyPurpose = "root" | "downloads" | "import";
