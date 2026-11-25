import type { LiveEventMediaServersMyAccessChangedEvent } from "./liveEventMediaServersMyAccessChangedEvent.ts";
import type { MediaserversMyAccessChanged } from "./mediaserversMyAccessChanged.ts";

export interface LiveEventMediaServersMyAccessChanged {
  event: LiveEventMediaServersMyAccessChangedEvent;
  data: MediaserversMyAccessChanged;
}
