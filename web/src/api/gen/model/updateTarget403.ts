import type { Problem } from "./problem.ts";
import type { UpdateTarget403Code } from "./updateTarget403Code.ts";

export type UpdateTarget403 = Problem & {
  code?: UpdateTarget403Code;
};
