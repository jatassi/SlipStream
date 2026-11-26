import type { PlaybackFactValue } from "./playbackFactValue.ts";
import type { PlaybackTargetRef } from "./playbackTargetRef.ts";
import type { PlaybackWorkLineState } from "./playbackWorkLineState.ts";

export interface PlaybackWorkLine {
  /** The Work's Content Type, whose noun the line uses */
  contentType: string;
  /** Absent unless state is not_proven: the first tracked fact not proven */
  fact?: PlaybackFactValue;
  /** The snapshot's Item, which Search for one scopes to */
  itemId: string;
  /** Absent unless state is not_in_profile */
  profile?: PlaybackTargetRef;
  state: PlaybackWorkLineState;
  title: string;
  workId: string;
}
