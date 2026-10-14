import type { GrabsChanged } from "./grabsChanged.ts";
import type { LiveEventGrabsChangedEvent } from "./liveEventGrabsChangedEvent.ts";

export interface LiveEventGrabsChanged {
  event: LiveEventGrabsChangedEvent;
  data: GrabsChanged;
}
