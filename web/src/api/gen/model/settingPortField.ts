import type { SettingPortFieldApplies } from "./settingPortFieldApplies.ts";
import type { SettingPortFieldKind } from "./settingPortFieldKind.ts";
import type { SettingPortFieldSource } from "./settingPortFieldSource.ts";

export interface SettingPortField {
  applies: SettingPortFieldApplies;
  default: number;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingPortFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  running: number;
  source: SettingPortFieldSource;
  value: number;
}
