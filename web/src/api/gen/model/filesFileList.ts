import type { FilesListedFile } from "./filesListedFile.ts";

export interface FilesFileList {
  /** Filling files first, in Profile order of the first Target each fills, then Kept files */
  files: FilesListedFile[];
  /** The Work's folder under its root, the one its naming template gives while it has no file */
  folder: string;
}
