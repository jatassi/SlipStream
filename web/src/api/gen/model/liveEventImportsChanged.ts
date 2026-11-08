import type { ImportsChanged } from "./importsChanged.ts";
import type { LiveEventImportsChangedEvent } from "./liveEventImportsChangedEvent.ts";

export interface LiveEventImportsChanged {
  event: LiveEventImportsChangedEvent;
  data: ImportsChanged;
}
