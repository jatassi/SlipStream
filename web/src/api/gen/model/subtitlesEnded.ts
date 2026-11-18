import type { SubtitlesEndedKind } from "./subtitlesEndedKind.ts";

export interface SubtitlesEnded {
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
