import type { GrabsSearchChanged } from "./grabsSearchChanged.ts";
import type { LiveEventGrabsSearchChangedEvent } from "./liveEventGrabsSearchChangedEvent.ts";

export interface LiveEventGrabsSearchChanged {
  event: LiveEventGrabsSearchChangedEvent;
  data: GrabsSearchChanged;
}
