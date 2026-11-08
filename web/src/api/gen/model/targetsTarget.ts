import type { TargetsFilter } from "./targetsFilter.ts";
import type { TargetsProfileRef } from "./targetsProfileRef.ts";
import type { TargetsScores } from "./targetsScores.ts";
import type { TargetsSize } from "./targetsSize.ts";
import type { TargetsTargetPropers } from "./targetsTargetPropers.ts";

export interface TargetsTarget {
  /** The other Targets every file of which also meets this one */
  contains: string[];
  createdAt: string;
  /** The first quality of the tier at which a file is final */
  cutoff: string;
  /** Every attribute that isn't Any, in attribute order */
  filters: TargetsFilter[];
  id: string;
  name: string;
  /** The Profiles holding it, by name */
  profiles: TargetsProfileRef[];
  /** prefer_and_upgrade ranks propers and repacks higher and has one replace a file of its Quality even past the cutoff. prefer_no_upgrade only ranks them higher. dont_prefer leaves them to the scores */
  propers: TargetsTargetPropers;
  scores: TargetsScores;
  /** Every accepted quality's size, in tier order */
  sizes: TargetsSize[];
  /** Quality keys, best tier first; a tier's qualities rank as equal */
  tiers: (string[] | null)[];
}
