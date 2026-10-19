import type { LiveEventWorksTypeChangedEvent } from "./liveEventWorksTypeChangedEvent.ts";
import type { WorksTypeChanged } from "./worksTypeChanged.ts";

export interface LiveEventWorksTypeChanged {
  event: LiveEventWorksTypeChangedEvent;
  data: WorksTypeChanged;
}
