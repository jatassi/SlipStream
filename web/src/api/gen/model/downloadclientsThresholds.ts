import type { SettingDurationField } from "./settingDurationField.ts";

export interface DownloadclientsThresholds {
  clientErrorFail: SettingDurationField;
  faultStall: SettingDurationField;
  metadataWait: SettingDurationField;
  unregisteredGap: SettingDurationField;
}
