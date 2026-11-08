import type { Problem } from "./problem.ts";
import type { SetWorkOptions404Code } from "./setWorkOptions404Code.ts";

export type SetWorkOptions404 = Problem & {
  code?: SetWorkOptions404Code;
};
