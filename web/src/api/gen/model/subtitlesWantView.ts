import type { SubtitlesLastSearch } from "./subtitlesLastSearch.ts";
import type { SubtitlesWantState } from "./subtitlesWantState.ts";
import type { TargetsWantedLanguage } from "./targetsWantedLanguage.ts";

export interface SubtitlesWantView {
  /** What its last search found; null until one ran */
  last: SubtitlesLastSearch | null;
  state: SubtitlesWantState;
  /** The want in the path's form: en, pt-BR, en-forced */
  want: string;
  wanted: TargetsWantedLanguage;
}
