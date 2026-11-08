import type { Problem } from "./problem.ts";
import type { SetWorkOptions409Code } from "./setWorkOptions409Code.ts";

export type SetWorkOptions409 = Problem & {
  code?: SetWorkOptions409Code;
};
