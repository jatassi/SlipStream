import type { SettingField } from "./settingField.ts";

export interface WorksTypeSettings {
  /** In the Content Type's order; empty when it has none */
  settings: SettingField[];
}
