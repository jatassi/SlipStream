import type { CheckSetupCode500Code } from "./checkSetupCode500Code.ts";
import type { Problem } from "./problem.ts";

export type CheckSetupCode500 = Problem & {
  code?: CheckSetupCode500Code;
};
