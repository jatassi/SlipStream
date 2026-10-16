import type { SettingDurationField } from "./settingDurationField.ts";

export interface TasksCadence {
  field: SettingDurationField;
  /** The task starts in the Maintenance window, every field.value seconds in whole days */
  inWindow: boolean;
  /** The cadence as the Tasks page says it, such as Every 8 h */
  words: string;
}
