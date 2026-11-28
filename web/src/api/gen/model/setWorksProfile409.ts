import type { Problem } from "./problem.ts";
import type { SetWorksProfile409Code } from "./setWorksProfile409Code.ts";
import type { TargetsImpact } from "./targetsImpact.ts";

export type SetWorksProfile409 = Problem & {
  code?: SetWorksProfile409Code;
  impact?: TargetsImpact;
};
