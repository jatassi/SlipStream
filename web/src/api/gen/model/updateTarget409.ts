import type { Problem } from "./problem.ts";
import type { TargetsImpact } from "./targetsImpact.ts";
import type { UpdateTarget409Code } from "./updateTarget409Code.ts";

export type UpdateTarget409 = Problem & {
  code?: UpdateTarget409Code;
  impact?: TargetsImpact;
  /** The Profiles that make another Target's files from it */
  profiles?: string[];
};
