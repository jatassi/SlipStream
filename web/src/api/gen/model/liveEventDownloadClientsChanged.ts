import type { DownloadclientsClientsChanged } from "./downloadclientsClientsChanged.ts";
import type { LiveEventDownloadClientsChangedEvent } from "./liveEventDownloadClientsChangedEvent.ts";

export interface LiveEventDownloadClientsChanged {
  event: LiveEventDownloadClientsChangedEvent;
  data: DownloadclientsClientsChanged;
}
