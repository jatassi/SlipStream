import type { HomeSettingsChanged } from "./homeSettingsChanged.ts";
import type { LiveEventHomeSettingsChangedEvent } from "./liveEventHomeSettingsChangedEvent.ts";

export interface LiveEventHomeSettingsChanged {
  event: LiveEventHomeSettingsChangedEvent;
  data: HomeSettingsChanged;
}
