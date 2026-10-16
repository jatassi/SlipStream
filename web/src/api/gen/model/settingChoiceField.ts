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
  options: SettingChoiceOption[];
  source: SettingChoiceFieldSource;
  value: string;
}
