import type { BackupSettingsInputBodyValues } from "./backupSettingsInputBodyValues.ts";

export interface BackupSettingsInputBody {
  /** Each setting's new value by key, all or none; null resets one to its default */
  values: BackupSettingsInputBodyValues;
}
