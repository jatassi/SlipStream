import type { GrabsBlocklistChanged } from "./grabsBlocklistChanged.ts";
import type { LiveEventGrabsBlocklistChangedEvent } from "./liveEventGrabsBlocklistChangedEvent.ts";

export interface LiveEventGrabsBlocklistChanged {
  event: LiveEventGrabsBlocklistChangedEvent;
  data: GrabsBlocklistChanged;
}
