import type { CreateCut409Code } from "./createCut409Code.ts";
import type { Problem } from "./problem.ts";

export type CreateCut409 = Problem & {
  code?: CreateCut409Code;
};
