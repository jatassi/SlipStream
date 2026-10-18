import type { Problem } from "./problem.ts";
import type { UpdateTarget409Code } from "./updateTarget409Code.ts";

export type UpdateTarget409 = Problem & {
  code?: UpdateTarget409Code;
};
