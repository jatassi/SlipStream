import type { TargetsEditions } from "./targetsEditions.ts";
import type { TargetsEntry } from "./targetsEntry.ts";

export interface TargetsNewProfile {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  editions: TargetsEditions;
  /**
   * @minLength 1
   * @maxLength 100
   */
  name: string;
  /** In Profile order; none is a Profile whose Works want nothing */
  targets: TargetsEntry[];
}
