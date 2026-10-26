import type { LiveEventTransfersChangedEvent } from "./liveEventTransfersChangedEvent.ts";
import type { TransfersChanged } from "./transfersChanged.ts";

export interface LiveEventTransfersChanged {
  event: LiveEventTransfersChangedEvent;
  data: TransfersChanged;
}
