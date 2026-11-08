import type { ImportsChosenFile } from "./importsChosenFile.ts";

export interface ImportIntoInputBody {
  /** @minItems 1 */
  files: ImportsChosenFile[];
  /** A Work in the pending import's root */
  workId: string;
}
