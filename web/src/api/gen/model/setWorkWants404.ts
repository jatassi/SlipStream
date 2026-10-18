import type { Problem } from "./problem.ts";
import type { SetWorkWants404Code } from "./setWorkWants404Code.ts";

export type SetWorkWants404 = Problem & {
  code?: SetWorkWants404Code;
};
