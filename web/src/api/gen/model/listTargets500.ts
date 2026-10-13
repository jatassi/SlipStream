import type { ListTargets500Code } from "./listTargets500Code.ts";
import type { Problem } from "./problem.ts";

export type ListTargets500 = Problem & {
  code?: ListTargets500Code;
};
