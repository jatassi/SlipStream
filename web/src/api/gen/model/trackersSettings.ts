import type { SettingField } from "./settingField.ts";

export interface TrackersSettings {
  /** The margin added to every rule's seeding time, and how long a torrent that owes may go without announcing before Health warns */
  settings: SettingField[];
}
