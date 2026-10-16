import type { SettingChoiceField } from "./settingChoiceField.ts";
import type { SettingCountField } from "./settingCountField.ts";
import type { SettingDurationField } from "./settingDurationField.ts";
import type { SettingWindowField } from "./settingWindowField.ts";

export interface TasksSchedule {
  headerDelay: SettingDurationField;
  jitterCap: SettingDurationField;
  jitterPercent: SettingCountField;
  overdue: SettingChoiceField;
  partialWarnIntervals: SettingCountField;
  runsKept: SettingCountField;
  window: SettingWindowField;
}
