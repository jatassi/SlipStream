import type { ImportsDeclineFileAction } from "./importsDeclineFileAction.ts";

export interface ImportsDeclineFile {
  /** Don't import the file */
  action: ImportsDeclineFileAction;
}
