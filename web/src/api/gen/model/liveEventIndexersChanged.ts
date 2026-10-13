import type { IndexersChanged } from "./indexersChanged.ts";
import type { LiveEventIndexersChangedEvent } from "./liveEventIndexersChangedEvent.ts";

export interface LiveEventIndexersChanged {
  event: LiveEventIndexersChangedEvent;
  data: IndexersChanged;
}
