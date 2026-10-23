import type { HealthNotReporting } from "./healthNotReporting.ts";
import type { HealthVolume } from "./healthVolume.ts";

export interface HealthVolumeList {
  /** Volumes with levels of their own that no Node reports now, by Node then mount */
  notReporting: HealthNotReporting[];
  /** Every volume a Node reports, by Node then mount */
  volumes: HealthVolume[];
}
