import type { FilesWaitingNeedCode } from "./filesWaitingNeedCode.ts";
import type { FilesWaitingNeedKind } from "./filesWaitingNeedKind.ts";

export interface FilesWaitingNeed {
  /** Why the Grab waits: node_away while its Node is out of step */
  code: FilesWaitingNeedCode;
  grabId: string;
  kind: FilesWaitingNeedKind;
}
