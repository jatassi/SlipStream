import type { SettingTimes } from "./settingTimes.ts";
import type { SettingWindowFieldApplies } from "./settingWindowFieldApplies.ts";
import type { SettingWindowFieldKind } from "./settingWindowFieldKind.ts";
import type { SettingWindowFieldSource } from "./settingWindowFieldSource.ts";

export interface SettingWindowField {
  applies: SettingWindowFieldApplies;
  default: SettingTimes;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingWindowFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  running: SettingTimes;
  source: SettingWindowFieldSource;
  value: SettingTimes;
  /** The time zone's name, such as Europe/London */
  zone: string;
}
