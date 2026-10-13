import type { LiveEventWorksContentTypesChangedEvent } from "./liveEventWorksContentTypesChangedEvent.ts";
import type { WorksContentTypesChanged } from "./worksContentTypesChanged.ts";

export interface LiveEventWorksContentTypesChanged {
  event: LiveEventWorksContentTypesChangedEvent;
  data: WorksContentTypesChanged;
}
