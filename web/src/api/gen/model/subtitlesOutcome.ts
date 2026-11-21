import type { SubtitlesOutcomeKind } from "./subtitlesOutcomeKind.ts";

export interface SubtitlesOutcome {
  kind: SubtitlesOutcomeKind;
  provider: string;
  /**
   * When a limited or capped provider takes calls again
   * @nullable
   */
  until: string | null;
}
