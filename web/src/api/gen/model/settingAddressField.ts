import type { SettingAddressFieldApplies } from "./settingAddressFieldApplies.ts";
import type { SettingAddressFieldKind } from "./settingAddressFieldKind.ts";
import type { SettingAddressFieldSource } from "./settingAddressFieldSource.ts";

export interface SettingAddressField {
  applies: SettingAddressFieldApplies;
  default: string;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingAddressFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  running: string;
  source: SettingAddressFieldSource;
  value: string;
}
