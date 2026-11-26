import type { LiveEventMediaServersPlaysChangedEvent } from "./liveEventMediaServersPlaysChangedEvent.ts";
import type { MediaserversPlaysChanged } from "./mediaserversPlaysChanged.ts";

export interface LiveEventMediaServersPlaysChanged {
  event: LiveEventMediaServersPlaysChangedEvent;
  data: MediaserversPlaysChanged;
}
