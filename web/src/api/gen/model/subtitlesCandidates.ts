import type { SubtitlesCandidate } from "./subtitlesCandidate.ts";
import type { SubtitlesCandidatesState } from "./subtitlesCandidatesState.ts";
import type { SubtitlesOutcome } from "./subtitlesOutcome.ts";

export interface SubtitlesCandidates {
  /** Best first */
  candidates: SubtitlesCandidate[];
  /** How each provider's search ended */
  outcomes: SubtitlesOutcome[];
  state: SubtitlesCandidatesState;
}
