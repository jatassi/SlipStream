import type { IndexersCatalogChanged } from "./indexersCatalogChanged.ts";
import type { LiveEventIndexersCatalogChangedEvent } from "./liveEventIndexersCatalogChangedEvent.ts";

export interface LiveEventIndexersCatalogChanged {
  event: LiveEventIndexersCatalogChangedEvent;
  data: IndexersCatalogChanged;
}
