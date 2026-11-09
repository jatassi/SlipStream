import type { Problem } from "./problem.ts";
import type { UpdateCustomFormat403Code } from "./updateCustomFormat403Code.ts";

export type UpdateCustomFormat403 = Problem & {
  code?: UpdateCustomFormat403Code;
};
