import type { SettingTimes } from "./settingTimes.ts";

export interface ConversionsHours {
  /** Whether the Node converts only inside the window */
  on: boolean;
  window: SettingTimes;
}
