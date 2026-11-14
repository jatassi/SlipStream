import type { SettingListFieldApplies } from "./settingListFieldApplies.ts";
import type { SettingListFieldItem } from "./settingListFieldItem.ts";
import type { SettingListFieldKind } from "./settingListFieldKind.ts";
import type { SettingListFieldSource } from "./settingListFieldSource.ts";

export interface SettingListField {
  applies: SettingListFieldApplies;
  default: string[];
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  /** What each entry is: an extension is lowercase letters and digits with no leading dot, such as mkv */
  item: SettingListFieldItem;
  key: string;
  kind: SettingListFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /**
   * The fewest entries the list takes
   * @minimum 0
   */
  min: number;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  running: string[];
  source: SettingListFieldSource;
  value: string[];
}
