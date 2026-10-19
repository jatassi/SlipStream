import type { SettingRateFieldApplies } from "./settingRateFieldApplies.ts";
import type { SettingRateFieldKind } from "./settingRateFieldKind.ts";
import type { SettingRateFieldSource } from "./settingRateFieldSource.ts";

export interface SettingRateField {
  applies: SettingRateFieldApplies;
  /** Bytes a second */
  default: number;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingRateFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /** Bytes a second */
  max: number;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  /** Bytes a second */
  running: number;
  source: SettingRateFieldSource;
  /** Bytes a second */
  value: number;
  /** The words for 0, such as No limit */
  zero?: string;
}
