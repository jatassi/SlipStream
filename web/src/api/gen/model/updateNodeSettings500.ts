import type { Problem } from "./problem.ts";
import type { UpdateNodeSettings500Code } from "./updateNodeSettings500Code.ts";

export type UpdateNodeSettings500 = Problem & {
  code?: UpdateNodeSettings500Code;
};
