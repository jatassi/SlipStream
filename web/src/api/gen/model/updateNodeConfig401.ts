import type { Problem } from "./problem.ts";
import type { UpdateNodeConfig401Code } from "./updateNodeConfig401Code.ts";

export type UpdateNodeConfig401 = Problem & {
  code?: UpdateNodeConfig401Code;
};
