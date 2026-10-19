import type { HealthChanged } from "./healthChanged.ts";
import type { LiveEventHealthChangedEvent } from "./liveEventHealthChangedEvent.ts";

export interface LiveEventHealthChanged {
  event: LiveEventHealthChangedEvent;
  data: HealthChanged;
}
