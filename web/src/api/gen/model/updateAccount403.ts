import type { Problem } from "./problem.ts";
import type { UpdateAccount403Code } from "./updateAccount403Code.ts";

export type UpdateAccount403 = Problem & {
  code?: UpdateAccount403Code;
};
