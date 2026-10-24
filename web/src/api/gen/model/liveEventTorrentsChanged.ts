import type { DownloadclientsChanged } from "./downloadclientsChanged.ts";
import type { LiveEventTorrentsChangedEvent } from "./liveEventTorrentsChangedEvent.ts";

export interface LiveEventTorrentsChanged {
  event: LiveEventTorrentsChangedEvent;
  data: DownloadclientsChanged;
}
