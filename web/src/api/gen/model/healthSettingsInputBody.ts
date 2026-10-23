import type { SettingAmount } from "./settingAmount.ts";

export interface HealthSettingsInputBody {
  /** null resets it to its default */
  errorBelow?: SettingAmount | null;
  /** null resets it to its default */
  warnBelow?: SettingAmount | null;
}
