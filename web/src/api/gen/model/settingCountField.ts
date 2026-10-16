import type { SettingCountFieldApplies } from "./settingCountFieldApplies.ts";
import type { SettingCountFieldKind } from "./settingCountFieldKind.ts";
import type { SettingCountFieldSource } from "./settingCountFieldSource.ts";

export interface SettingCountField {
  applies: SettingCountFieldApplies;
  default: number;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** Why the minimum protects someone outside the install */
  floor?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingCountFieldKind;
  label: string;
  max: number;
  min: number;
  source: SettingCountFieldSource;
  unit: string;
  value: number;
}
