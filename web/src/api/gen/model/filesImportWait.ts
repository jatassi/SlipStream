import type { FilesSpace } from "./filesSpace.ts";

export interface FilesImportWait {
  rootId: string;
  /** For no_space, the bytes free and the bytes the file needs; null when a clone found the volume full or for root_not_ready */
  space: FilesSpace | null;
}
