import type { TargetsBackfillRest } from "./targetsBackfillRest.ts";

export interface TargetsBackfill {
  /** The Works to convert */
  convert: string[];
  /** The Profile new Works take */
  default: string;
  /** Where the other Works go */
  rest: TargetsBackfillRest;
}
