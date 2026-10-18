import type { SettingDurationField } from "./settingDurationField.ts";

export interface NodesSettings {
  joinCodeLifetime: SettingDurationField;
  offlineGrace: SettingDurationField;
}
