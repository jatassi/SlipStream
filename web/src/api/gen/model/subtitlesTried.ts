import type { SubtitlesTriedOutcome } from "./subtitlesTriedOutcome.ts";

export interface SubtitlesTried {
  at: string;
  /**
   * Its timing match, in percent
   * @nullable
   */
  match: number | null;
  outcome: SubtitlesTriedOutcome;
}
