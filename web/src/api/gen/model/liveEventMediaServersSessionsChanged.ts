import type { LiveEventMediaServersSessionsChangedEvent } from "./liveEventMediaServersSessionsChangedEvent.ts";
import type { MediaserversSessionsChanged } from "./mediaserversSessionsChanged.ts";

export interface LiveEventMediaServersSessionsChanged {
  event: LiveEventMediaServersSessionsChangedEvent;
  data: MediaserversSessionsChanged;
}
