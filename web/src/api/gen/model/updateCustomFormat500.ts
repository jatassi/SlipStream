import type { Problem } from "./problem.ts";
import type { UpdateCustomFormat500Code } from "./updateCustomFormat500Code.ts";

export type UpdateCustomFormat500 = Problem & {
  code?: UpdateCustomFormat500Code;
};
