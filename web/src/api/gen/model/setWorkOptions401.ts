import type { Problem } from "./problem.ts";
import type { SetWorkOptions401Code } from "./setWorkOptions401Code.ts";

export type SetWorkOptions401 = Problem & {
  code?: SetWorkOptions401Code;
};
