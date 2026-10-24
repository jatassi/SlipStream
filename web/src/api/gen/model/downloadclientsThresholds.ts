import type { SettingDurationField } from "./settingDurationField.ts";

export interface DownloadclientsThresholds {
  faultStall: SettingDurationField;
  unregisteredGap: SettingDurationField;
}
