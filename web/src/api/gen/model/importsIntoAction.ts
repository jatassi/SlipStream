import type { ImportsChosenFile } from "./importsChosenFile.ts";
import type { ImportsIntoActionAction } from "./importsIntoActionAction.ts";

export interface ImportsIntoAction {
  action: ImportsIntoActionAction;
  /** @minItems 1 */
  files: ImportsChosenFile[];
  /** A Work in the Import's root */
  workId: string;
}
