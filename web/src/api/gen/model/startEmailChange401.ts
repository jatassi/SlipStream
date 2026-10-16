import type { Problem } from "./problem.ts";
import type { StartEmailChange401Code } from "./startEmailChange401Code.ts";

export type StartEmailChange401 = Problem & {
  code?: StartEmailChange401Code;
};
