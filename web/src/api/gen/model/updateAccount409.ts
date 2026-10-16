import type { Problem } from "./problem.ts";
import type { UpdateAccount409Code } from "./updateAccount409Code.ts";

export type UpdateAccount409 = Problem & {
  code?: UpdateAccount409Code;
};
