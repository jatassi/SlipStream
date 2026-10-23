import type { HealthVolumeLevel } from "./healthVolumeLevel.ts";
import type { SettingSpaceField } from "./settingSpaceField.ts";

export interface HealthVolume {
  /** Its own error level, the default while it follows the global one */
  errorBelow: SettingSpaceField;
  /** The error level it follows, in bytes */
  errorBytes: number;
  /** The bytes SlipStream may write */
  free: number;
  /**
   * Its Storage item's level, null while it has none
   * @nullable
   */
  level: HealthVolumeLevel;
  /** Its Node is in step, so its figures are current; else they read as of sampledAt */
  live: boolean;
  /** As its Node names it, such as D:\ or /mnt/media */
  mount: string;
  /** Such as D: (Media) */
  name: string;
  nodeId: string;
  nodeName: string;
  sampledAt: string;
  /** Its error level is at or above its warning level, so it goes straight to error */
  skipsWarning: boolean;
  total: number;
  /** Its own warning level, the default while it follows the global one */
  warnBelow: SettingSpaceField;
  /** The warning level it follows, in bytes */
  warnBytes: number;
}
