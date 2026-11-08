import type { LiveEventTrackersSeedingChangedEvent } from "./liveEventTrackersSeedingChangedEvent.ts";
import type { TrackersSeedingChanged } from "./trackersSeedingChanged.ts";

export interface LiveEventTrackersSeedingChanged {
  event: LiveEventTrackersSeedingChangedEvent;
  data: TrackersSeedingChanged;
}
