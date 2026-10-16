import type { Problem } from "./problem.ts";
import type { UpdateNodeConfig404Code } from "./updateNodeConfig404Code.ts";

export type UpdateNodeConfig404 = Problem & {
  code?: UpdateNodeConfig404Code;
};
