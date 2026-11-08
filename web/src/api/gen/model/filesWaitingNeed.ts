import type { FilesWaitingNeedCode } from "./filesWaitingNeedCode.ts";
import type { FilesWaitingNeedKind } from "./filesWaitingNeedKind.ts";

export interface FilesWaitingNeed {
  /** Why the Grab waits: node_away while its Node, or the Node of the client its fetch goes to, is out of step, client_away while its Torrent's client isn't responding, refused the sign-in or needs attention */
  code: FilesWaitingNeedCode;
  grabId: string;
  kind: FilesWaitingNeedKind;
}
