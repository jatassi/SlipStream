import type { SettingIPFieldApplies } from "./settingIPFieldApplies.ts";
import type { SettingIPFieldFamily } from "./settingIPFieldFamily.ts";
import type { SettingIPFieldKind } from "./settingIPFieldKind.ts";
import type { SettingIPFieldSource } from "./settingIPFieldSource.ts";

export interface SettingIPField {
  applies: SettingIPFieldApplies;
  /**
   * Always null: the setting is empty by default
   * @nullable
   */
  default: string | null;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** Which addresses the setting takes: any IP address, or IPv6 only */
  family: SettingIPFieldFamily;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingIPFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  /**
   * The address, or null when it is empty
   * @nullable
   */
  running: string | null;
  source: SettingIPFieldSource;
  /**
   * The address, or null when it is empty
   * @nullable
   */
  value: string | null;
}
