import type { SettingField } from "./settingField.ts";

export interface ImportsImportingSettings {
  /** The video file extensions, the smallest video file, then the runtime under which a video is a sample */
  settings: SettingField[];
}
