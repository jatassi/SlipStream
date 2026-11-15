import type { IndexersRSSChanged } from "./indexersRSSChanged.ts";
import type { LiveEventIndexersRssChangedEvent } from "./liveEventIndexersRssChangedEvent.ts";

export interface LiveEventIndexersRssChanged {
  event: LiveEventIndexersRssChangedEvent;
  data: IndexersRSSChanged;
}
