import type { Problem } from "./problem.ts";
import type { UpdateNodeConverting401Code } from "./updateNodeConverting401Code.ts";

export type UpdateNodeConverting401 = Problem & {
  code?: UpdateNodeConverting401Code;
};
