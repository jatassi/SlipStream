import type { SettingField } from "./settingField.ts";

export interface SettingsOutputBody {
  /** Wait before sending a new Health item, Give up on a message after, Warn when a connection has failed for, and Keep notifications for */
  settings: SettingField[];
}
