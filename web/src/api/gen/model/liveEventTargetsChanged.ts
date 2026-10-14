import type { LiveEventTargetsChangedEvent } from "./liveEventTargetsChangedEvent.ts";
import type { TargetsChanged } from "./targetsChanged.ts";

export interface LiveEventTargetsChanged {
  event: LiveEventTargetsChangedEvent;
  data: TargetsChanged;
}
