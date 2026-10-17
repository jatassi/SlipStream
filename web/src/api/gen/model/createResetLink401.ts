import type { CreateResetLink401Code } from "./createResetLink401Code.ts";
import type { Problem } from "./problem.ts";

export type CreateResetLink401 = Problem & {
  code?: CreateResetLink401Code;
};
