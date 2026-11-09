import type { SettingField } from "./settingField.ts";

export interface ImportsSettings {
  /** The stop share, then the files each Node probes at once */
  settings: SettingField[];
}
