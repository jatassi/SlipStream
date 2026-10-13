import type { LiveEventRootsChangedEvent } from "./liveEventRootsChangedEvent.ts";
import type { RootsChanged } from "./rootsChanged.ts";

export interface LiveEventRootsChanged {
  event: LiveEventRootsChangedEvent;
  data: RootsChanged;
}
