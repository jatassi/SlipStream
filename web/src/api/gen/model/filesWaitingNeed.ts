import type { FilesClientError } from "./filesClientError.ts";
import type { FilesWaitingNeedCode } from "./filesWaitingNeedCode.ts";
import type { FilesWaitingNeedKind } from "./filesWaitingNeedKind.ts";

export interface FilesWaitingNeed {
  /** The client's error, for client_error; null otherwise */
  clientError: FilesClientError | null;
  /** Why the Grab waits: node_away while its Node, or the Node of the client its fetch goes to, is out of step, client_away while its Torrent's client isn't responding, refused the sign-in or needs attention, client_off while the client it is pinned to is off or deleted, or its Node doesn't hold Downloading. no_space while its root has too little free space for a file, root_not_ready while its root isn't ready, probe_timeout while reading a file timed out three times, and client_error while its external client reports an error */
  code: FilesWaitingNeedCode;
  grabId: string;
  kind: FilesWaitingNeedKind;
}
