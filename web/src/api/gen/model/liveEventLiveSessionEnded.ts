import type { LiveEventLiveSessionEndedEvent } from "./liveEventLiveSessionEndedEvent.ts";
import type { LiveSessionEnded } from "./liveSessionEnded.ts";

export interface LiveEventLiveSessionEnded {
  event: LiveEventLiveSessionEndedEvent;
  data: LiveSessionEnded;
}
