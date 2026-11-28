import type { DeleteCustomFormat409Code } from "./deleteCustomFormat409Code.ts";
import type { Problem } from "./problem.ts";
import type { TargetsImpact } from "./targetsImpact.ts";

export type DeleteCustomFormat409 = Problem & {
  code?: DeleteCustomFormat409Code;
  impact?: TargetsImpact;
};
