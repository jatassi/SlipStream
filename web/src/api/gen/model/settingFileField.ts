import type { SettingFileFieldApplies } from "./settingFileFieldApplies.ts";
import type { SettingFileFieldKind } from "./settingFileFieldKind.ts";
import type { SettingFileFieldSource } from "./settingFileFieldSource.ts";

export interface SettingFileField {
  applies: SettingFileFieldApplies;
  /** Empty runs the program SlipStream bundles */
  default: string;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingFileFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /** The Node the path is on, which checks that the program is there and runs */
  nodeId: string;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  running: string;
  source: SettingFileFieldSource;
  /** SlipStream's support doesn't cover a program set here */
  unsupported: boolean;
  value: string;
}
