import type { ListTargets401Code } from "./listTargets401Code.ts";
import type { Problem } from "./problem.ts";

export type ListTargets401 = Problem & {
  code?: ListTargets401Code;
};
