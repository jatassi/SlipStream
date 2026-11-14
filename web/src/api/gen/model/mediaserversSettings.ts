import type { SettingField } from "./settingField.ts";

export interface MediaserversSettings {
  /** Wait before sending, Check sessions every, Mark unhealthy after, Keep play history for, and the Plex sign-in's renewal schedule and margin */
  settings: SettingField[];
}
