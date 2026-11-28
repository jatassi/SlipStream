import type { CreateTarget422Code } from "./createTarget422Code.ts";
import type { Problem } from "./problem.ts";
import type { TargetsProblem } from "./targetsProblem.ts";

export type CreateTarget422 = Problem & {
  code?: CreateTarget422Code;
  /** @nullable */
  problems?: TargetsProblem[] | null;
};
