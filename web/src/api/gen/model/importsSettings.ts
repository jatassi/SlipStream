import type { SettingField } from "./settingField.ts";

export interface ImportsSettings {
  /** The stop share, then how many files each Node reads at once */
  settings: SettingField[];
}
