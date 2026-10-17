import type { CreateResetLink500Code } from "./createResetLink500Code.ts";
import type { Problem } from "./problem.ts";

export type CreateResetLink500 = Problem & {
  code?: CreateResetLink500Code;
};
