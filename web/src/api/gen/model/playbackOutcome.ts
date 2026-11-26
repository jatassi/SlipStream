import type { PlaybackBlocked } from "./playbackBlocked.ts";
import type { PlaybackEvidence } from "./playbackEvidence.ts";
import type { PlaybackOutcomeKind } from "./playbackOutcomeKind.ts";
import type { PlaybackSuggestion } from "./playbackSuggestion.ts";
import type { PlaybackTargetRef } from "./playbackTargetRef.ts";
import type { PlaybackVersion } from "./playbackVersion.ts";
import type { PlaybackWorkLine } from "./playbackWorkLine.ts";

export interface PlaybackOutcome {
  /** Absent unless kind is version, suggest or existing: one per proven value */
  evidence?: PlaybackEvidence[];
  kind: PlaybackOutcomeKind;
  /** Absent unless kind is unproven: the first dimension with no proven value */
  missing?: PlaybackBlocked;
  /** @minimum 1 */
  plays: number;
  /** Absent unless kind is suggest */
  suggestion?: PlaybackSuggestion;
  /** Absent unless kind is existing */
  target?: PlaybackTargetRef;
  /** Absent unless kind is version */
  version?: PlaybackVersion;
  /** Absent unless kind is existing */
  works?: PlaybackWorkLine[];
}
