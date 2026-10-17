import type { LiveAccessChanged } from "./liveAccessChanged.ts";
import type { LiveEventLiveAccessChangedEvent } from "./liveEventLiveAccessChangedEvent.ts";

export interface LiveEventLiveAccessChanged {
  event: LiveEventLiveAccessChangedEvent;
  data: LiveAccessChanged;
}
