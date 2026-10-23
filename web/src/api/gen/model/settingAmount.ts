import type { SettingAmountUnit } from "./settingAmountUnit.ts";

export interface SettingAmount {
  /** @minimum 0 */
  amount: number;
  unit: SettingAmountUnit;
}
