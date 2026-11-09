import type { CreateCustomFormat500Code } from "./createCustomFormat500Code.ts";
import type { Problem } from "./problem.ts";

export type CreateCustomFormat500 = Problem & {
  code?: CreateCustomFormat500Code;
};
