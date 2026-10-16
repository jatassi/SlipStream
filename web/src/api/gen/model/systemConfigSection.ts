import type { SettingField } from "./settingField.ts";
import type { SystemConfigSectionId } from "./systemConfigSectionId.ts";

export interface SystemConfigSection {
  fields: SettingField[];
  id: SystemConfigSectionId;
}
