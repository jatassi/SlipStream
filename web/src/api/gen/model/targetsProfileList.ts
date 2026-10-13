import type { TargetsProfile } from "./targetsProfile.ts";

export interface TargetsProfileList {
  /** By Content Type, then name */
  profiles: TargetsProfile[];
}
