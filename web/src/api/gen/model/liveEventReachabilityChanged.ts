import type { LiveEventReachabilityChangedEvent } from "./liveEventReachabilityChangedEvent.ts";
import type { ReachabilityChanged } from "./reachabilityChanged.ts";

export interface LiveEventReachabilityChanged {
  event: LiveEventReachabilityChangedEvent;
  data: ReachabilityChanged;
}
