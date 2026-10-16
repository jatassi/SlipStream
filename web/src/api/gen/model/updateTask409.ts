import type { Problem } from "./problem.ts";
import type { UpdateTask409Code } from "./updateTask409Code.ts";

export type UpdateTask409 = Problem & {
  code?: UpdateTask409Code;
};
