import type { LiveEventSubtitlesChangedEvent } from "./liveEventSubtitlesChangedEvent.ts";
import type { SubtitlesChanged } from "./subtitlesChanged.ts";

export interface LiveEventSubtitlesChanged {
  event: LiveEventSubtitlesChangedEvent;
  data: SubtitlesChanged;
}
