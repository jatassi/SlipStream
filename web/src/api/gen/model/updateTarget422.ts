import type { Problem } from "./problem.ts";
import type { TargetsProblem } from "./targetsProblem.ts";
import type { UpdateTarget422Code } from "./updateTarget422Code.ts";

export type UpdateTarget422 = Problem & {
  code?: UpdateTarget422Code;
  /** @nullable */
  problems?: TargetsProblem[] | null;
};
