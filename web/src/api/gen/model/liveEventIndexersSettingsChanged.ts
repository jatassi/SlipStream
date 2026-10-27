import type { IndexersSettingsChanged } from "./indexersSettingsChanged.ts";
import type { LiveEventIndexersSettingsChangedEvent } from "./liveEventIndexersSettingsChangedEvent.ts";

export interface LiveEventIndexersSettingsChanged {
  event: LiveEventIndexersSettingsChangedEvent;
  data: IndexersSettingsChanged;
}
