import type { TargetsWantedLanguageAudio } from "./targetsWantedLanguageAudio.ts";
import type { TargetsWantedLanguageHearingImpaired } from "./targetsWantedLanguageHearingImpaired.ts";

export interface TargetsWantedLanguage {
  /** When the language is wanted. always is whatever the audio, and no_audio is only when no audio track is in the language */
  audio: TargetsWantedLanguageAudio;
  /** Whether only the lines in another language, such as signs, are wanted, not every line of dialogue */
  forced: boolean;
  /** How hearing-impaired subtitles rank. avoid uses one only when nothing else fits, and prefer chooses one first when one fits */
  hearingImpaired: TargetsWantedLanguageHearingImpaired;
  /**
   * An ISO 639 code, or pt-BR for Brazilian Portuguese
   * @minLength 2
   * @maxLength 5
   */
  language: string;
}
