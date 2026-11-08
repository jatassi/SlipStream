import type { LibraryScanSettingsInputBodyValues } from "./libraryScanSettingsInputBodyValues.ts";

export interface LibraryScanSettingsInputBody {
  /** Each setting's new value by key, all or none; null resets one to its default */
  values: LibraryScanSettingsInputBodyValues;
}
