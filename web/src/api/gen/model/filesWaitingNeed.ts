import type { FilesWaitingNeedKind } from "./filesWaitingNeedKind.ts";

export interface FilesWaitingNeed {
  /** Why the Grab waits, such as needs_transfer */
  code: string;
  grabId: string;
  kind: FilesWaitingNeedKind;
}
