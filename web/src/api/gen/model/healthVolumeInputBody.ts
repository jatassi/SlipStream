import type { SettingAmount } from "./settingAmount.ts";

export interface HealthVolumeInputBody {
  /** null makes it follow the global level */
  errorBelow?: SettingAmount | null;
  /** @minLength 1 */
  mount: string;
  nodeId: string;
  /** null makes it follow the global level */
  warnBelow?: SettingAmount | null;
}
