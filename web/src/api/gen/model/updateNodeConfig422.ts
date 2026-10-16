import type { Problem } from "./problem.ts";
import type { UpdateNodeConfig422Code } from "./updateNodeConfig422Code.ts";

export type UpdateNodeConfig422 = Problem & {
  code?: UpdateNodeConfig422Code;
};
