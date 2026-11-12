import type { ImportsNotThisReleaseFileAction } from "./importsNotThisReleaseFileAction.ts";

export interface ImportsNotThisReleaseFile {
  /** Put the download's Release on the Blocklist and search again; a Grab's Import only */
  action: ImportsNotThisReleaseFileAction;
}
