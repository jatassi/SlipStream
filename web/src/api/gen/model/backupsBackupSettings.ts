import type { SettingField } from "./settingField.ts";

export interface BackupsBackupSettings {
  /** How often SlipStream backs up, how long it keeps Backups, when Health warns, how long a copy waits and how often copies are checked */
  settings: SettingField[];
}
