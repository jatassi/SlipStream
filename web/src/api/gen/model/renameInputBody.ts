import type { FilesConfirmedWork } from "./filesConfirmedWork.ts";

export interface RenameInputBody {
  /**
   * Each Work to rename as its plan showed
   * @minItems 1
   */
  works: FilesConfirmedWork[];
}
