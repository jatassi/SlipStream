import type { SettingCountFieldApplies } from "./settingCountFieldApplies.ts";
import type { SettingCountFieldKind } from "./settingCountFieldKind.ts";
import type { SettingCountFieldSource } from "./settingCountFieldSource.ts";
import type { SettingNoun } from "./settingNoun.ts";

export interface SettingCountField {
  applies: SettingCountFieldApplies;
  default: number;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** Why the minimum protects someone outside the install */
  floor?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingCountFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  max: number;
  min: number;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  running: number;
  source: SettingCountFieldSource;
  unit: SettingNoun;
  value: number;
  /** The words for 0, such as No limit */
  zero?: string;
}
