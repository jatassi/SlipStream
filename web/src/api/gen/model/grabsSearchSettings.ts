import type { SettingField } from "./settingField.ts";

export interface GrabsSearchSettings {
  /** Backlog search and how often it runs, RSS sync, how long a search waits for each Indexer and when SlipStream stops searching again after failed Grabs */
  settings: SettingField[];
}
