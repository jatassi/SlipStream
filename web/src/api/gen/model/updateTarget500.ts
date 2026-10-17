import type { Problem } from "./problem.ts";
import type { UpdateTarget500Code } from "./updateTarget500Code.ts";

export type UpdateTarget500 = Problem & {
  code?: UpdateTarget500Code;
};
