/**
 * Each setting's new value by key, all or none; null resets one to its default. A new language or region makes every Work due at the next Metadata refresh.
 */
export type MetadataSettingsChangeValues = { [key: string]: unknown };
