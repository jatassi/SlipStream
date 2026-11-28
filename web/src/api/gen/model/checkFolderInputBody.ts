import type { CheckFolderInputBodyPurpose } from "./checkFolderInputBodyPurpose.ts";

export interface CheckFolderInputBody {
  /**
   * @minLength 1
   * @maxLength 4096
   */
  path: string;
  /** root: a library root's rules, on a Storage Node; downloads: a downloads folder's, writable, on no network share and apart from every root on the Node; import: a folder to import from, inside one of the Node's library roots or download folders; backups: a folder for Backups, writable and apart from every root and downloads folder on the Node; converting: a Node's scratch folder for Conversions, writable and apart from every root, downloads folder and the Backups folder on the Node */
  purpose: CheckFolderInputBodyPurpose;
  /** For root: the root being re-homed, which the folder may overlap */
  rootId?: string;
}
