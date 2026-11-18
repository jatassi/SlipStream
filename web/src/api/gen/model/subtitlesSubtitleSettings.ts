import type { SettingField } from "./settingField.ts";
import type { SubtitlesLead } from "./subtitlesLead.ts";
import type { SubtitlesProfileLanguages } from "./subtitlesProfileLanguages.ts";

export interface SubtitlesSubtitleSettings {
  lead: SubtitlesLead;
  /** Each Profile with a Wanted Language, by name */
  profiles: SubtitlesProfileLanguages[];
  /** Subtitle search, upgrades and how often it runs, then the settings under Details */
  settings: SettingField[];
}
