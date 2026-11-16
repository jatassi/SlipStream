import type { GrabsQueueChanged } from "./grabsQueueChanged.ts";
import type { LiveEventGrabsQueueChangedEvent } from "./liveEventGrabsQueueChangedEvent.ts";

export interface LiveEventGrabsQueueChanged {
  event: LiveEventGrabsQueueChangedEvent;
  data: GrabsQueueChanged;
}
