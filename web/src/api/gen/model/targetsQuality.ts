import type { TargetsQualityKey } from "./targetsQualityKey.ts";
import type { TargetsQualityResolution } from "./targetsQualityResolution.ts";
import type { TargetsQualitySource } from "./targetsQualitySource.ts";

export interface TargetsQuality {
  /** A cinema recording: kept out of a Target unless an Admin adds it */
  cinema: boolean;
  /**
   * A new Target's minimum size, in MiB per minute
   * @minimum 0
   */
  floor: number;
  key: TargetsQualityKey;
  name: string;
  /**
   * Null for cinema sources
   * @nullable
   */
  resolution: TargetsQualityResolution;
  source: TargetsQualitySource;
}
