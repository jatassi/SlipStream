import type { Problem } from "./problem.ts";
import type { UpdateRootSettings500Code } from "./updateRootSettings500Code.ts";

export type UpdateRootSettings500 = Problem & {
  code?: UpdateRootSettings500Code;
};
