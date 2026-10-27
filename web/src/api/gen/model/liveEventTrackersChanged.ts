import type { IndexersTrackerChanged } from "./indexersTrackerChanged.ts";
import type { LiveEventTrackersChangedEvent } from "./liveEventTrackersChangedEvent.ts";

export interface LiveEventTrackersChanged {
  event: LiveEventTrackersChangedEvent;
  data: IndexersTrackerChanged;
}
