export interface TargetsRenamedFile {
  fileId: string;
  /** The file's path in its root now */
  from: string;
  /** The file's path in its root after the change */
  to: string;
}
