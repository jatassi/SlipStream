import type { LiveEventMediaServersChangedEvent } from "./liveEventMediaServersChangedEvent.ts";
import type { MediaserversChanged } from "./mediaserversChanged.ts";

export interface LiveEventMediaServersChanged {
  event: LiveEventMediaServersChangedEvent;
  data: MediaserversChanged;
}
