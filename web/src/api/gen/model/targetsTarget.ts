import type { TargetsFilter } from "./targetsFilter.ts";
import type { TargetsProfileRef } from "./targetsProfileRef.ts";
import type { TargetsSize } from "./targetsSize.ts";

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
  /** Every accepted quality's size, in tier order */
  sizes: TargetsSize[];
  /** Quality keys, best tier first; a tier's qualities rank as equal */
  tiers: (string[] | null)[];
}
