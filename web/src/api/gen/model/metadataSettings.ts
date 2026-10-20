import type { SettingField } from "./settingField.ts";

export interface MetadataSettings {
  /** Language and region first, then the cache, OMDb's ratings and the artwork store */
  settings: SettingField[];
}
