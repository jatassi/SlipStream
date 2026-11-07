import type { IndexersStatusChanged } from "./indexersStatusChanged.ts";
import type { LiveEventIndexersStatusChangedEvent } from "./liveEventIndexersStatusChangedEvent.ts";

export interface LiveEventIndexersStatusChanged {
  event: LiveEventIndexersStatusChangedEvent;
  data: IndexersStatusChanged;
}
