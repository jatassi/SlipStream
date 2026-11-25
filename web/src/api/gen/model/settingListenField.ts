import type { SettingListenAt } from "./settingListenAt.ts";
import type { SettingListenFieldApplies } from "./settingListenFieldApplies.ts";
import type { SettingListenFieldKind } from "./settingListenFieldKind.ts";
import type { SettingListenFieldSource } from "./settingListenFieldSource.ts";

export interface SettingListenField {
  applies: SettingListenFieldApplies;
  default: SettingListenAt;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingListenFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  running: SettingListenAt;
  source: SettingListenFieldSource;
  value: SettingListenAt;
}
