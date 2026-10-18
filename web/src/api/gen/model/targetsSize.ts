import type { TargetsSizeQuality } from "./targetsSizeQuality.ts";

export interface TargetsSize {
  /**
   * MiB per minute; null is no limit
   * @minimum 0
   * @nullable
   */
  max: number | null;
  /**
   * MiB per minute of the runtime the file covers
   * @minimum 0
   */
  min: number;
  quality: TargetsSizeQuality;
}
