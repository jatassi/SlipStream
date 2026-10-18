import type { TargetsImpact } from "./targetsImpact.ts";
import type { TargetsProfile } from "./targetsProfile.ts";

export interface TargetsProfileChange {
  impact: TargetsImpact;
  profile: TargetsProfile;
}
