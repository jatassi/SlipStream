import type { Problem } from "./problem.ts";
import type { UpdateNodeConfig409Code } from "./updateNodeConfig409Code.ts";

export type UpdateNodeConfig409 = Problem & {
  code?: UpdateNodeConfig409Code;
};
