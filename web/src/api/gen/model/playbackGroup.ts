import type { PlaybackBlocked } from "./playbackBlocked.ts";
import type { PlaybackOutcome } from "./playbackOutcome.ts";
import type { PlaybackTargetRef } from "./playbackTargetRef.ts";

export interface PlaybackGroup {
  /** The dimensions the player's line names: the first no Target fixes, else at most two */
  cause: PlaybackBlocked[];
  /** In dimension table order */
  dimensions: PlaybackBlocked[];
  /** Largest first */
  outcomes: PlaybackOutcome[];
  /** @minimum 1 */
  plays: number;
  /** As the server gave them */
  reasons: string[];
  /** The Targets the transcoded files met when they played */
  targets: PlaybackTargetRef[];
}
