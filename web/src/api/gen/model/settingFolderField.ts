import type { SettingFolderFieldApplies } from "./settingFolderFieldApplies.ts";
import type { SettingFolderFieldKind } from "./settingFolderFieldKind.ts";
import type { SettingFolderFieldPurpose } from "./settingFolderFieldPurpose.ts";
import type { SettingFolderFieldSource } from "./settingFolderFieldSource.ts";

export interface SettingFolderField {
  applies: SettingFolderFieldApplies;
  default: string;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingFolderFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /** The Node the path is on, which browses and judges it */
  nodeId: string;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  /** The rules the Node judges the folder by, as checkFolder names them */
  purpose: SettingFolderFieldPurpose;
  running: string;
  source: SettingFolderFieldSource;
  value: string;
}
