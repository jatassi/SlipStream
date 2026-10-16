import type { CreateResetLink403Code } from "./createResetLink403Code.ts";
import type { Problem } from "./problem.ts";

export type CreateResetLink403 = Problem & {
  code?: CreateResetLink403Code;
};
