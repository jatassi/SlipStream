import type { TargetsEditions } from "./targetsEditions.ts";
import type { TargetsEntry } from "./targetsEntry.ts";
import type { TargetsEstimate } from "./targetsEstimate.ts";

export interface TargetsProfileDetail {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /** Whether a Work added with no Profile uses it */
  default: boolean;
  editions: TargetsEditions;
  /** One per Target, in Profile order. A Target no file of known runtime fills yet has no estimate */
  estimates: TargetsEstimate[];
  id: string;
  name: string;
  /** In Profile order: a file is named after the first Target it is the file for */
  targets: TargetsEntry[];
  /**
   * The Works that use it
   * @minimum 0
   */
  works: number;
}
