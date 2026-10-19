import type { SettingChoiceField } from "./settingChoiceField.ts";
import type { SettingCountField } from "./settingCountField.ts";
import type { SettingDurationField } from "./settingDurationField.ts";
import type { SettingPortField } from "./settingPortField.ts";
import type { SettingSpaceField } from "./settingSpaceField.ts";
import type { SettingSwitchField } from "./settingSwitchField.ts";
import type { SettingWindowField } from "./settingWindowField.ts";

export type SettingField =
  | SettingDurationField
  | SettingCountField
  | SettingChoiceField
  | SettingWindowField
  | SettingPortField
  | SettingSwitchField
  | SettingSpaceField;
