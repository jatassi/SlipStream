import type { TrackersSettingsChangeValues } from "./trackersSettingsChangeValues.ts";

export interface TrackersSettingsChange {
  /** Each setting's new value by key, all or none; null resets one to its default */
  values: TrackersSettingsChangeValues;
}
