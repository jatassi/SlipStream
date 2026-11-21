import type { SubtitlesCandidate } from "./subtitlesCandidate.ts";
import type { SubtitlesCandidatesState } from "./subtitlesCandidatesState.ts";
import type { SubtitlesOutcome } from "./subtitlesOutcome.ts";

export interface SubtitlesCandidates {
  /** Best first */
  candidates: SubtitlesCandidate[];
  /** How the search ended on each provider, in search order. SlipStream doesn't search a paused provider, and its outcome shows the pause */
  outcomes: SubtitlesOutcome[];
  state: SubtitlesCandidatesState;
}
