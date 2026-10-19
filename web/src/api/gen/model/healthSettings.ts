import type { SettingSpaceField } from "./settingSpaceField.ts";

export interface HealthSettings {
  errorBelow: SettingSpaceField;
  warnBelow: SettingSpaceField;
}
