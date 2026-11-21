import type { TargetsWantedLanguage } from "./targetsWantedLanguage.ts";

export interface SubtitlesProfileLanguages {
  contentType: string;
  id: string;
  /** In the Admin's order */
  languages: TargetsWantedLanguage[];
  name: string;
}
