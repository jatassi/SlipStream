import type { LiveEventSubtitlesProvidersChangedEvent } from "./liveEventSubtitlesProvidersChangedEvent.ts";
import type { SubtitlesProvidersChanged } from "./subtitlesProvidersChanged.ts";

export interface LiveEventSubtitlesProvidersChanged {
  event: LiveEventSubtitlesProvidersChangedEvent;
  data: SubtitlesProvidersChanged;
}
