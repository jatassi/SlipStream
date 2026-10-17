import type { Problem } from "./problem.ts";
import type { StartEmailChange500Code } from "./startEmailChange500Code.ts";

export type StartEmailChange500 = Problem & {
  code?: StartEmailChange500Code;
};
