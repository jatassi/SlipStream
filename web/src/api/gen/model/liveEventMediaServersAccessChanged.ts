import type { LiveEventMediaServersAccessChangedEvent } from "./liveEventMediaServersAccessChangedEvent.ts";
import type { MediaserversAccessChanged } from "./mediaserversAccessChanged.ts";

export interface LiveEventMediaServersAccessChanged {
  event: LiveEventMediaServersAccessChangedEvent;
  data: MediaserversAccessChanged;
}
