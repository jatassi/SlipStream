import type { SettingDurationFieldApplies } from "./settingDurationFieldApplies.ts";
import type { SettingDurationFieldKind } from "./settingDurationFieldKind.ts";
import type { SettingDurationFieldSource } from "./settingDurationFieldSource.ts";

export interface SettingDurationField {
  applies: SettingDurationFieldApplies;
  /** Seconds */
  default: number;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** Why the minimum protects someone outside the install */
  floor?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingDurationFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /** Seconds */
  max: number;
  /** Seconds */
  min: number;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  /** Seconds */
  running: number;
  source: SettingDurationFieldSource;
  /** Seconds: a second, minute, hour or day */
  step: number;
  /** Seconds */
  value: number;
}
