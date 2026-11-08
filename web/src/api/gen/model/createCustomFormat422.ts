import type { CreateCustomFormat422Code } from "./createCustomFormat422Code.ts";
import type { Problem } from "./problem.ts";

export type CreateCustomFormat422 = Problem & {
  code?: CreateCustomFormat422Code;
};
