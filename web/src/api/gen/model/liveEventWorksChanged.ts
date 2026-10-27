import type { LiveEventWorksChangedEvent } from "./liveEventWorksChangedEvent.ts";
import type { WorksChanged } from "./worksChanged.ts";

export interface LiveEventWorksChanged {
  event: LiveEventWorksChangedEvent;
  data: WorksChanged;
}
