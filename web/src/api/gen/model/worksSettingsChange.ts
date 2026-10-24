import type { WorksSettingsChangeValues } from "./worksSettingsChangeValues.ts";

export interface WorksSettingsChange {
  /** Each setting's new value by key, all or none; null resets one to its default */
  values: WorksSettingsChangeValues;
}
