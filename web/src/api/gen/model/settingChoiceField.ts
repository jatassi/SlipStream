import type { SettingChoiceFieldApplies } from "./settingChoiceFieldApplies.ts";
import type { SettingChoiceFieldKind } from "./settingChoiceFieldKind.ts";
import type { SettingChoiceFieldSource } from "./settingChoiceFieldSource.ts";
import type { SettingChoiceOption } from "./settingChoiceOption.ts";

export interface SettingChoiceField {
  applies: SettingChoiceFieldApplies;
  default: string;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingChoiceFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  options: SettingChoiceOption[];
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  running: string;
  source: SettingChoiceFieldSource;
  value: string;
}
