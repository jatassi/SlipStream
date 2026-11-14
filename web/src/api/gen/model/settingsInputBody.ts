import type { SettingsInputBodyValues } from "./settingsInputBodyValues.ts";

export interface SettingsInputBody {
  /** Each setting's new value by key, all or none; null resets one to its default */
  values: SettingsInputBodyValues;
}
