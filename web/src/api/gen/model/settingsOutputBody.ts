import type { SettingField } from "./settingField.ts";

export interface SettingsOutputBody {
  /** Wait before sending a new Health item, Give up on a message after and Warn when a connection has failed for */
  settings: SettingField[];
}
