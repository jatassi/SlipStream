import type { LiveEventMediaServersItemsFoundEvent } from "./liveEventMediaServersItemsFoundEvent.ts";
import type { MediaserversFound } from "./mediaserversFound.ts";

export interface LiveEventMediaServersItemsFound {
  event: LiveEventMediaServersItemsFoundEvent;
  data: MediaserversFound;
}
