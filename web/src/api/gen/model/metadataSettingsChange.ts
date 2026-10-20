import type { MetadataSettingsChangeValues } from "./metadataSettingsChangeValues.ts";

export interface MetadataSettingsChange {
  /** Each setting's new value by key, all or none; null resets one to its default. A new language or region makes every Work due at the next Metadata refresh. */
  values: MetadataSettingsChangeValues;
}
