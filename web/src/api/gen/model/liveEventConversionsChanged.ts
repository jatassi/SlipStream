import type { ConversionsChanged } from "./conversionsChanged.ts";
import type { LiveEventConversionsChangedEvent } from "./liveEventConversionsChangedEvent.ts";

export interface LiveEventConversionsChanged {
  event: LiveEventConversionsChangedEvent;
  data: ConversionsChanged;
}
