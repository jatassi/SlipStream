import type { LiveEventNodesFencedEvent } from "./liveEventNodesFencedEvent.ts";
import type { NodesFenced } from "./nodesFenced.ts";

export interface LiveEventNodesFenced {
  event: LiveEventNodesFencedEvent;
  data: NodesFenced;
}
