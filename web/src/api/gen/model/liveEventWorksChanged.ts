import type { LiveEventWorksChangedEvent } from "./liveEventWorksChangedEvent.ts";
import type { WorksWorkChanged } from "./worksWorkChanged.ts";

export interface LiveEventWorksChanged {
  event: LiveEventWorksChangedEvent;
  data: WorksWorkChanged;
}
