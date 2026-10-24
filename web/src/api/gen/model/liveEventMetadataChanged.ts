import type { LiveEventMetadataChangedEvent } from "./liveEventMetadataChangedEvent.ts";
import type { MetadataChanged } from "./metadataChanged.ts";

export interface LiveEventMetadataChanged {
  event: LiveEventMetadataChangedEvent;
  data: MetadataChanged;
}
