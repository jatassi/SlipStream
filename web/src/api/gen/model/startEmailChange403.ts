import type { Problem } from "./problem.ts";
import type { StartEmailChange403Code } from "./startEmailChange403Code.ts";

export type StartEmailChange403 = Problem & {
  code?: StartEmailChange403Code;
};
