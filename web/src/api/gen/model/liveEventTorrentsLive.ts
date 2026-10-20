import type { DownloadclientsLiveTick } from "./downloadclientsLiveTick.ts";
import type { LiveEventTorrentsLiveEvent } from "./liveEventTorrentsLiveEvent.ts";

export interface LiveEventTorrentsLive {
  event: LiveEventTorrentsLiveEvent;
  data: DownloadclientsLiveTick;
}
