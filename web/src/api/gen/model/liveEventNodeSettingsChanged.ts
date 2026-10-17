import type { LiveEventNodeSettingsChangedEvent } from "./liveEventNodeSettingsChangedEvent.ts";
import type { NodesSettingsChanged } from "./nodesSettingsChanged.ts";

export interface LiveEventNodeSettingsChanged {
  event: LiveEventNodeSettingsChangedEvent;
  data: NodesSettingsChanged;
}
