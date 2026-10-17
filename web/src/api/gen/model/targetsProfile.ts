import type { TargetsEditions } from "./targetsEditions.ts";
import type { TargetsEntry } from "./targetsEntry.ts";

export interface TargetsProfile {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /** Whether a Work added with no Profile takes it */
  default: boolean;
  editions: TargetsEditions;
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
