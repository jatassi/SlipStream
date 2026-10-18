import type { TargetsEditions } from "./targetsEditions.ts";
import type { TargetsEntry } from "./targetsEntry.ts";
import type { TargetsEstimate } from "./targetsEstimate.ts";

export interface TargetsProfileDetail {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /** Whether a Work added with no Profile takes it */
  default: boolean;
  editions: TargetsEditions;
  /** One per Target, in Profile order. A Target no file of known runtime fills yet has no estimate */
  estimates: TargetsEstimate[];
  id: string;
  name: string;
  /** In Profile order: a file is named after the first it fills */
  targets: TargetsEntry[];
  /**
   * The Works wanting from it
   * @minimum 0
   */
  works: number;
}
