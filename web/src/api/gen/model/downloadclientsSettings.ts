import type { SettingCountField } from "./settingCountField.ts";
import type { SettingDurationField } from "./settingDurationField.ts";
import type { SettingRateField } from "./settingRateField.ts";
import type { SettingSwitchField } from "./settingSwitchField.ts";

export interface DownloadclientsSettings {
  activeChecks: SettingCountField;
  activeDownloads: SettingCountField;
  connections: SettingCountField;
  downloadLimit: SettingRateField;
  stalledAfter: SettingDurationField;
  uploadLimit: SettingRateField;
  utp: SettingSwitchField;
}
