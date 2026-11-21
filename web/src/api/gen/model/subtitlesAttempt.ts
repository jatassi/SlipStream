import type { SubtitlesAttemptStep } from "./subtitlesAttemptStep.ts";
import type { SubtitlesEnded } from "./subtitlesEnded.ts";

export interface SubtitlesAttempt {
  /** How it ended; null until it finished */
  outcome: SubtitlesEnded | null;
  /** @nullable */
  provider: string | null;
  step: SubtitlesAttemptStep;
  /**
   * A refused upload SlipStream can keep anyway
   * @nullable
   */
  uploadId: string | null;
}
