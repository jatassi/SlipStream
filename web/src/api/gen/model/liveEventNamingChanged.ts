import type { LiveEventNamingChangedEvent } from "./liveEventNamingChangedEvent.ts";
import type { NamingChanged } from "./namingChanged.ts";

export interface LiveEventNamingChanged {
  event: LiveEventNamingChangedEvent;
  data: NamingChanged;
}
