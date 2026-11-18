import type { SubtitlesCandidateReasonsItem } from "./subtitlesCandidateReasonsItem.ts";
import type { SubtitlesTried } from "./subtitlesTried.ts";

export interface SubtitlesCandidate {
  /** It can't be picked: it holds image subtitles, or its provider is paused. A candidate marked bad can be picked, which undoes the Mark bad */
  disabled: boolean;
  key: string;
  provider: string;
  /** The ranking steps it meets */
  reasons: SubtitlesCandidateReasonsItem[];
  release: string;
  /** Its earlier try for this file and want */
  tried: SubtitlesTried | null;
}
