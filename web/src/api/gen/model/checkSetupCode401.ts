import type { CheckSetupCode401Code } from "./checkSetupCode401Code.ts";
import type { Problem } from "./problem.ts";

export type CheckSetupCode401 = Problem & {
  code?: CheckSetupCode401Code;
};
