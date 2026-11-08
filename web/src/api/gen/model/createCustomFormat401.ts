import type { CreateCustomFormat401Code } from "./createCustomFormat401Code.ts";
import type { Problem } from "./problem.ts";

export type CreateCustomFormat401 = Problem & {
  code?: CreateCustomFormat401Code;
};
