import type { MediaserversSessionView } from "./mediaserversSessionView.ts";

export interface MediaserversSessions {
  /** false while the Connect from Node is offline, when sessions is empty */
  reachable: boolean;
  sessions: MediaserversSessionView[];
}
