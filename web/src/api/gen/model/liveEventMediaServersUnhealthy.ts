import type { LiveEventMediaServersUnhealthyEvent } from "./liveEventMediaServersUnhealthyEvent.ts";
import type { MediaserversWentUnhealthy } from "./mediaserversWentUnhealthy.ts";

export interface LiveEventMediaServersUnhealthy {
  event: LiveEventMediaServersUnhealthyEvent;
  data: MediaserversWentUnhealthy;
}
