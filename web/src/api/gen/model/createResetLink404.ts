import type { CreateResetLink404Code } from "./createResetLink404Code.ts";
import type { Problem } from "./problem.ts";

export type CreateResetLink404 = Problem & {
  code?: CreateResetLink404Code;
};
