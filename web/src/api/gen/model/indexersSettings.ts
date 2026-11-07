import type { SettingField } from "./settingField.ts";

export interface IndexersSettings {
  /** How long every request to an Indexer may fail before Health warns */
  settings: SettingField[];
}
