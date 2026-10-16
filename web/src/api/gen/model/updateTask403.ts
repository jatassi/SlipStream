import type { Problem } from "./problem.ts";
import type { UpdateTask403Code } from "./updateTask403Code.ts";

export type UpdateTask403 = Problem & {
  code?: UpdateTask403Code;
};
