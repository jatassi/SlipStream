import type { CreateCustomFormat400Code } from "./createCustomFormat400Code.ts";
import type { Problem } from "./problem.ts";

export type CreateCustomFormat400 = Problem & {
  code?: CreateCustomFormat400Code;
};
