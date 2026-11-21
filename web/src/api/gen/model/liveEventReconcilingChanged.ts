import type { LiveEventReconcilingChangedEvent } from "./liveEventReconcilingChangedEvent.ts";
import type { ReconcilingChanged } from "./reconcilingChanged.ts";

export interface LiveEventReconcilingChanged {
  event: LiveEventReconcilingChangedEvent;
  data: ReconcilingChanged;
}
