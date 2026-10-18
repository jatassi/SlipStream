import type { TargetsEditions } from "./targetsEditions.ts";
import type { TargetsWantsFrom } from "./targetsWantsFrom.ts";

export interface TargetsWants {
  /** The effective Edition list */
  editions: TargetsEditions;
  /** Whether the Edition list is the Profile's or the Work's own */
  from: TargetsWantsFrom;
  profileId: string;
  profileName: string;
}
