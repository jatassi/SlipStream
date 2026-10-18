import type { Problem } from "./problem.ts";
import type { SetWorkWants401Code } from "./setWorkWants401Code.ts";

export type SetWorkWants401 = Problem & {
  code?: SetWorkWants401Code;
};
