import type { FirstrunChanged } from "./firstrunChanged.ts";
import type { LiveEventFirstrunChangedEvent } from "./liveEventFirstrunChangedEvent.ts";

export interface LiveEventFirstrunChanged {
  event: LiveEventFirstrunChangedEvent;
  data: FirstrunChanged;
}
