import type { LiveEventVolumesChangedEvent } from "./liveEventVolumesChangedEvent.ts";
import type { RootsVolumesChanged } from "./rootsVolumesChanged.ts";

export interface LiveEventVolumesChanged {
  event: LiveEventVolumesChangedEvent;
  data: RootsVolumesChanged;
}
