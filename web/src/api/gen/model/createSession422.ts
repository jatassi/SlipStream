import type { CreateSession422Code } from "./createSession422Code.ts";
import type { Problem } from "./problem.ts";

export type CreateSession422 = Problem & {
  code?: CreateSession422Code;
};
