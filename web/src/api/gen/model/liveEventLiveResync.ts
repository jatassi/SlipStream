import type { LiveEventLiveResyncEvent } from "./liveEventLiveResyncEvent.ts";
import type { LiveResync } from "./liveResync.ts";

export interface LiveEventLiveResync {
  event: LiveEventLiveResyncEvent;
  data: LiveResync;
}
