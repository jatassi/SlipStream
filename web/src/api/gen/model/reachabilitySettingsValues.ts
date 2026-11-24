import type { ReachabilitySettingsValuesValues } from "./reachabilitySettingsValuesValues.ts";

export interface ReachabilitySettingsValues {
  /** Each setting's new value by key, all or none; null resets one to its default */
  values: ReachabilitySettingsValuesValues;
}
