import type { LiveEventSubtitlesAttemptEvent } from "./liveEventSubtitlesAttemptEvent.ts";
import type { SubtitlesAttempted } from "./subtitlesAttempted.ts";

export interface LiveEventSubtitlesAttempt {
  event: LiveEventSubtitlesAttemptEvent;
  data: SubtitlesAttempted;
}
