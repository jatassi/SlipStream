import type { SettingField } from "./settingField.ts";

export interface TrackersSettings {
  /** The extra seeding time added to every rule's seeding time, and how long a torrent at hit and run risk may go without announcing before Health warns */
  settings: SettingField[];
}
