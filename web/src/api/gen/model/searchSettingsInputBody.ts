import type { SearchSettingsInputBodyValues } from "./searchSettingsInputBodyValues.ts";

export interface SearchSettingsInputBody {
  /** Each setting's new value by key, all or none; null resets one to its default */
  values: SearchSettingsInputBodyValues;
}
