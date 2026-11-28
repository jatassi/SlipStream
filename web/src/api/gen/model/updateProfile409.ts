import type { Problem } from "./problem.ts";
import type { TargetsImpact } from "./targetsImpact.ts";
import type { UpdateProfile409Code } from "./updateProfile409Code.ts";

export type UpdateProfile409 = Problem & {
  code?: UpdateProfile409Code;
  impact?: TargetsImpact;
  /** The Profiles that make another Target's files from it */
  profiles?: string[];
};
