import type { SettingField } from "./settingField.ts";

export interface IndexersSettings {
  /** How long a search waits for each Indexer, and how long every request to an Indexer may fail before Health warns */
  settings: SettingField[];
}
