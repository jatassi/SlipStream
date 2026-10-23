import type { SettingAmount } from "./settingAmount.ts";

export interface HealthNotReporting {
  /** Its own error level, null while it follows the global one */
  errorBelow: SettingAmount | null;
  mount: string;
  nodeId: string;
  nodeName: string;
  /** Its own warning level, null while it follows the global one */
  warnBelow: SettingAmount | null;
}
