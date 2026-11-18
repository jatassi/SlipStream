import type { SubtitlesEndedKind } from "./subtitlesEndedKind.ts";
import type { SubtitlesOutcome } from "./subtitlesOutcome.ts";

export interface SubtitlesEnded {
  /** Why the provider didn't serve the download, on unfetched */
  answer: SubtitlesOutcome | null;
  kind: SubtitlesEndedKind;
  /**
   * The timing match, in percent
   * @nullable
   */
  match: number | null;
  /**
   * The timing match a subtitle needs, in percent
   * @nullable
   */
  threshold: number | null;
}
