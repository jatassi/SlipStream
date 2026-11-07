import type { IndexersSettingsChangeValues } from "./indexersSettingsChangeValues.ts";

export interface IndexersSettingsChange {
  /** Each setting's new value by key, all or none; null resets one to its default */
  values: IndexersSettingsChangeValues;
}
