import type { Problem } from "./problem.ts";
import type { UpdateNodeConfig500Code } from "./updateNodeConfig500Code.ts";

export type UpdateNodeConfig500 = Problem & {
  code?: UpdateNodeConfig500Code;
};
