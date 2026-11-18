import type { SubtitlesCandidateReasonsItem } from "./subtitlesCandidateReasonsItem.ts";
import type { SubtitlesTried } from "./subtitlesTried.ts";

export interface SubtitlesCandidate {
  /** It can't be picked: image subtitles, marked bad, or its provider is out */
  disabled: boolean;
  key: string;
  provider: string;
  /** The ranking steps it meets */
  reasons: SubtitlesCandidateReasonsItem[];
  release: string;
  /** Its earlier try for this file and want */
  tried: SubtitlesTried | null;
}
