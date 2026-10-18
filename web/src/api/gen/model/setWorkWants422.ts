import type { Problem } from "./problem.ts";
import type { SetWorkWants422Code } from "./setWorkWants422Code.ts";

export type SetWorkWants422 = Problem & {
  code?: SetWorkWants422Code;
};
