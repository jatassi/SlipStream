import type { LiveEventWorksRemovedEvent } from "./liveEventWorksRemovedEvent.ts";
import type { WorksRemoved } from "./worksRemoved.ts";

export interface LiveEventWorksRemoved {
  event: LiveEventWorksRemovedEvent;
  data: WorksRemoved;
}
