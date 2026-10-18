import type { TargetsFilter } from "./targetsFilter.ts";
import type { TargetsNewTargetCutoff } from "./targetsNewTargetCutoff.ts";
import type { TargetsSize } from "./targetsSize.ts";

export interface TargetsNewTarget {
  /** Any quality of the tier at which a file is final */
  cutoff: TargetsNewTargetCutoff;
  /**
   * An attribute left out is Any
   * @nullable
   */
  filters?: TargetsFilter[] | null;
  /**
   * @minLength 1
   * @maxLength 100
   */
  name: string;
  /**
   * An accepted quality left out takes its floor and no maximum
   * @nullable
   */
  sizes?: TargetsSize[] | null;
  /**
   * Quality keys, best tier first; a tier's qualities rank as equal
   * @minItems 1
   */
  tiers: (string[] | null)[];
}
