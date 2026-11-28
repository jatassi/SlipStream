import type { TargetsConversionSpec } from "./targetsConversionSpec.ts";
import type { TargetsFilter } from "./targetsFilter.ts";
import type { TargetsNewScores } from "./targetsNewScores.ts";
import type { TargetsNewTargetCutoff } from "./targetsNewTargetCutoff.ts";
import type { TargetsNewTargetPropers } from "./targetsNewTargetPropers.ts";
import type { TargetsSize } from "./targetsSize.ts";

export interface TargetsNewTarget {
  /** Its Conversion settings; null or left out for none */
  conversion?: TargetsConversionSpec;
  /** Any quality of the tier at which a file is final */
  cutoff: TargetsNewTargetCutoff;
  /**
   * An attribute left out is Any
   * @nullable
   */
  filters?: TargetsFilter[] | null;
  /**
   * How long an automatic grab waits after a Release's publish time for a better one, in whole minutes; left out, 0 grabs at once. Changing only the Hold changes no Work
   * @minimum 0
   * @maximum 604800
   */
  holdSeconds?: number;
  /**
   * @minLength 1
   * @maxLength 100
   */
  name: string;
  /** prefer_and_upgrade ranks propers and repacks higher and has one replace a file of its Quality even past the cutoff. prefer_no_upgrade only ranks them higher. dont_prefer leaves them to the scores */
  propers?: TargetsNewTargetPropers;
  /**
   * A replace only: the Works whose files the Target's older Conversion settings made, to make again
   * @nullable
   */
  remake?: string[] | null;
  /** Left out, every score is 0 and the increment is 1 */
  scores?: TargetsNewScores;
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
