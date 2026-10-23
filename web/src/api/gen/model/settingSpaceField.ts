import type { SettingAmount } from "./settingAmount.ts";
import type { SettingSpaceFieldApplies } from "./settingSpaceFieldApplies.ts";
import type { SettingSpaceFieldKind } from "./settingSpaceFieldKind.ts";
import type { SettingSpaceFieldSource } from "./settingSpaceFieldSource.ts";

export interface SettingSpaceField {
  applies: SettingSpaceFieldApplies;
  default: SettingAmount;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingSpaceFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /** The largest amount in bytes: 1 PB */
  maxBytes: number;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  running: SettingAmount;
  source: SettingSpaceFieldSource;
  value: SettingAmount;
}
