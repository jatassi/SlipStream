import type { LiveEventMediaServersRecoveredEvent } from "./liveEventMediaServersRecoveredEvent.ts";
import type { MediaserversRecovered } from "./mediaserversRecovered.ts";

export interface LiveEventMediaServersRecovered {
  event: LiveEventMediaServersRecoveredEvent;
  data: MediaserversRecovered;
}
