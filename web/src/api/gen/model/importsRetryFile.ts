import type { ImportsRetryFileAction } from "./importsRetryFileAction.ts";

export interface ImportsRetryFile {
  /** Try the file again */
  action: ImportsRetryFileAction;
}
