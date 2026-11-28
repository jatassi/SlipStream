import type { Problem } from "./problem.ts";
import type { UpdateNodeConverting404Code } from "./updateNodeConverting404Code.ts";

export type UpdateNodeConverting404 = Problem & {
  code?: UpdateNodeConverting404Code;
};
