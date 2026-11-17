import type { HistoryChanged } from "./historyChanged.ts";
import type { LiveEventHistoryChangedEvent } from "./liveEventHistoryChangedEvent.ts";

export interface LiveEventHistoryChanged {
  event: LiveEventHistoryChangedEvent;
  data: HistoryChanged;
}
