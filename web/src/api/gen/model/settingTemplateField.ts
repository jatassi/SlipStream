import type { SettingTemplateFieldApplies } from "./settingTemplateFieldApplies.ts";
import type { SettingTemplateFieldKind } from "./settingTemplateFieldKind.ts";
import type { SettingTemplateFieldSource } from "./settingTemplateFieldSource.ts";

export interface SettingTemplateField {
  applies: SettingTemplateFieldApplies;
  default: string;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** The template field whose tokens the control offers */
  field: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingTemplateFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  /** The text holds no tokens */
  plain: boolean;
  source: SettingTemplateFieldSource;
  value: string;
}
