import type { MediaserversSessionView } from "./mediaserversSessionView.ts";

export interface MediaserversSessions {
  /** false while the reaching Node is offline, when sessions is empty */
  reachable: boolean;
  sessions: MediaserversSessionView[];
}
